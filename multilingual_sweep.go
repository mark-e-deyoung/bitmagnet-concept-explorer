package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/mark-e-deyoung/bitmagnet-concept-explorer/internal/contracts"
	"github.com/mark-e-deyoung/bitmagnet-concept-explorer/internal/deadline"
)

// The admitted worst case is 12 Bitmagnet terms at fanout 4 and 8s/request
// (24s), followed by 3 SearXNG requests at 8s each with 2s between starts
// (28s). The 55s deadline leaves 3s for serialization and scheduling.
const (
	sweepExecutionDeadline = 55 * time.Second
	// HTTP WriteTimeout starts before the request body is read. Leave room for
	// the 15s ReadTimeout plus the 55s sweep budget and response serialization.
	sweepHTTPWriteTimeout = 75 * time.Second
	sweepClientTimeout    = deadline.ClientTimeout
	maxConcurrentSweeps   = 1
)

// At most one whole sweep may occupy the local backends at once. This is
// separate from the existing per-source provider admission limits.
var activeSweepSlots = make(chan struct{}, maxConcurrentSweeps)

// Sweep requests are one-shot and bounded. There is no daemon, crawler,
// scheduler, provider account integration, autonomous model invocation, or CPE
// authority mutation. The graph input is a digest-pinned RESEARCH preview.
type SweepRequest struct {
	Query               string       `json:"query"`
	ConceptIDs          []string     `json:"concept_ids,omitempty"`
	Languages           []string     `json:"languages,omitempty"`
	IncludeSearxng      bool         `json:"include_searxng,omitempty"`
	ExternalConsent     bool         `json:"external_consent,omitempty"`
	MaxBitmagnetQueries int          `json:"max_bitmagnet_queries,omitempty"`
	MaxSearxngQueries   int          `json:"max_searxng_queries,omitempty"`
	Filters             QueryFilters `json:"filters,omitempty"`
}
type SweepQuery struct {
	Term       string `json:"term"`
	Language   string `json:"language"`
	ConceptID  string `json:"concept_id,omitempty"`
	Provenance string `json:"provenance"`
}
type SweepPlan struct {
	Version          string       `json:"version"`
	Authority        string       `json:"authority"`
	ProjectionDigest string       `json:"projection_digest,omitempty"`
	ReadOnly         bool         `json:"read_only"`
	Bitmagnet        []SweepQuery `json:"bitmagnet_queries"`
	Searxng          []SweepQuery `json:"searxng_queries"`
	ExternalAdmitted bool         `json:"external_admitted"`
	Omitted          []string     `json:"omitted"`
	Warnings         []string     `json:"warnings"`
}
type SweepExternalObservation struct {
	Term        string                `json:"term"`
	Language    string                `json:"language"`
	Hits        []ExternalObservation `json:"hits"`
	Suggestions []string              `json:"suggestions,omitempty"`
}
type SweepResult struct {
	State             string                     `json:"state"`
	DeadlineSeconds   int                        `json:"deadline_seconds"`
	Plan              SweepPlan                  `json:"plan"`
	Bitmagnet         *Execution                 `json:"bitmagnet,omitempty"`
	External          []SweepExternalObservation `json:"external"`
	Failures          []string                   `json:"failures"`
	Evidence          []SweepDiscoveryEvidence   `json:"evidence_refs"`
	ResearchProposals []SweepDiscoveryDraft      `json:"research_proposals"`
	CPEWrite          bool                       `json:"cpe_write"`
	Warning           string                     `json:"warning"`
}

func sweepLanguage(v string) bool {
	if len(v) < 2 || len(v) > 20 {
		return false
	}
	parts := strings.Split(v, "-")
	for i, p := range parts {
		if len(p) < 2 || len(p) > 8 {
			return false
		}
		for _, r := range p {
			if r > 127 || !unicode.IsLetter(r) {
				if i == 0 || r < '0' || r > '9' {
					return false
				}
			}
		}
	}
	return true
}
func safeExternalSweepTerm(v string) bool {
	if strings.TrimSpace(v) == "" || strings.HasPrefix(strings.TrimSpace(v), ":") || utf8.RuneCountInString(v) > 160 {
		return false
	}
	for _, r := range v {
		if unicode.IsControl(r) || r == '!' || r == '\\' {
			return false
		}
	}
	return true
}
func privateSearxngSweepAllowed() bool {
	if os.Getenv("BCE_SEARXNG_SWEEP_ALLOWED") != "true" {
		return false
	}
	raw, e := searXNGEndpoint()
	if e != nil {
		return false
	}
	u, e := url.Parse(raw)
	if e != nil {
		return false
	}
	if u.Hostname() == "localhost" {
		return true
	}
	ip := net.ParseIP(u.Hostname())
	return ip != nil && (ip.IsPrivate() || ip.IsLoopback())
}
func makeSweepPlan(r SweepRequest) (SweepPlan, error) {
	base := strings.TrimSpace(r.Query)
	if base == "" || utf8.RuneCountInString(base) > 512 {
		return SweepPlan{}, errors.New("sweep query must contain 1-512 characters")
	}
	if len(r.ConceptIDs) > 8 || len(r.Languages) > 12 {
		return SweepPlan{}, errors.New("too many concept/language selections")
	}
	if r.IncludeSearxng && !r.ExternalConsent {
		return SweepPlan{}, errors.New("external SearXNG sweeps require per-request explicit consent")
	}
	if r.MaxBitmagnetQueries < 0 || r.MaxBitmagnetQueries > 12 || r.MaxSearxngQueries < 0 || r.MaxSearxngQueries > 3 {
		return SweepPlan{}, errors.New("sweep budgets outside maximum bounds")
	}
	if e := validateView(Request{Query: base, Filters: r.Filters}); e != nil {
		return SweepPlan{}, e
	}
	maxLocal := r.MaxBitmagnetQueries
	if maxLocal == 0 {
		maxLocal = 8
	}
	maxExternal := r.MaxSearxngQueries
	if maxExternal == 0 {
		maxExternal = 3
	}
	wanted := map[string]bool{}
	for _, lang := range r.Languages {
		lang = strings.ToLower(strings.TrimSpace(lang))
		if !sweepLanguage(lang) || wanted[lang] {
			return SweepPlan{}, errors.New("invalid or duplicate requested language")
		}
		wanted[lang] = true
	}
	plan := SweepPlan{Version: "sweep-research-v0", Authority: "digest-pinned-preview-not-cpe-accepted", ReadOnly: true,
		Bitmagnet: []SweepQuery{}, Searxng: []SweepQuery{}, Omitted: []string{},
		Warnings: []string{"Lexicalizations are search candidates, not equivalent concept assertions", "No-hit, failed source and omitted query never establish semantic absence", "SearXNG may forward terms to third-party search engines; obey its configured limits and provider policies; never submit local corpus data"},
	}
	forms := []SweepQuery{{Term: base, Language: "und", Provenance: "user-direct-literal"}}
	if len(r.ConceptIDs) > 0 {
		cat, e := readConceptCatalog()
		if e != nil {
			return SweepPlan{}, e
		}
		plan.ProjectionDigest = cat.Digest
		ids := map[string]bool{}
		for _, id := range r.ConceptIDs {
			if ids[id] {
				return SweepPlan{}, errors.New("duplicate selected concept ID")
			}
			ids[id] = true
		}
		for _, id := range r.ConceptIDs {
			found := false
			for _, node := range cat.Concepts {
				if node.ID != id {
					continue
				}
				found = true
				// The preview label has no declared language, so do not invent one.
				forms = append(forms, SweepQuery{Term: node.Label, Language: "und", ConceptID: id, Provenance: "pinned-preview:" + cat.SnapshotID})
				for _, lex := range node.Lexicalizations {
					lang := strings.ToLower(lex.Language)
					if len(wanted) > 0 && !wanted[lang] {
						continue
					}
					forms = append(forms, SweepQuery{Term: lex.Term, Language: lang, ConceptID: id, Provenance: "pinned-preview:" + cat.SnapshotID})
				}
				break
			}
			if !found {
				return SweepPlan{}, errors.New("concept not found in pinned preview")
			}
		}
	}
	// One Bitmagnet FTS expression per distinct literal term, regardless of
	// how many CPE lexicalization records refer to it.
	seenLocal := map[string]bool{}
	for _, f := range forms {
		v := strings.TrimSpace(f.Term)
		if v == "" || utf8.RuneCountInString(v) > 512 {
			return SweepPlan{}, errors.New("invalid sweep term")
		}
		if isPinnedProjectionProvenance(f.Provenance) {
			if err := validateBitmagnetSafeLiteral(f.Term); err != nil {
				return SweepPlan{}, fmt.Errorf("CPE-generated lexicalization is not admitted as a native FTS literal: %w", err)
			}
		}
		key := strings.ToLower(v)
		if seenLocal[key] {
			continue
		}
		seenLocal[key] = true
		if len(plan.Bitmagnet) >= maxLocal {
			plan.Omitted = append(plan.Omitted, "bitmagnet budget: "+f.Language)
			continue
		}
		f.Term = v
		plan.Bitmagnet = append(plan.Bitmagnet, f)
	}
	if r.IncludeSearxng {
		plan.ExternalAdmitted = privateSearxngSweepAllowed()
		if !plan.ExternalAdmitted {
			return SweepPlan{}, errors.New("unattended external sweep requires operator-attested private SearXNG /search endpoint; public instances are not automatically swept")
		}
		seenExternal := map[string]bool{}
		for _, f := range forms {
			f.Term = strings.TrimSpace(f.Term)
			// Filter by language on the backend only when the lexical form has
			// explicit language provenance. 'und' is sent without a language hint.
			if !safeExternalSweepTerm(f.Term) {
				plan.Omitted = append(plan.Omitted, "external unsafe query syntax omitted")
				continue
			}
			key := strings.ToLower(f.Term) + "\x00" + f.Language
			if seenExternal[key] {
				continue
			}
			seenExternal[key] = true
			if len(plan.Searxng) >= maxExternal {
				plan.Omitted = append(plan.Omitted, "searxng budget: "+f.Language)
				continue
			}
			plan.Searxng = append(plan.Searxng, f)
		}
	}
	sort.Strings(plan.Omitted)
	return plan, nil
}
func sweepOpaqueRef(parts ...string) string {
	hash := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return "sha256:" + hex.EncodeToString(hash[:])
}
func addSweepEvidence(r *SweepResult) {
	r.Evidence = []SweepDiscoveryEvidence{}
	seen := map[string]bool{}
	// The ancestry identifier intentionally denotes the whole unqualified
	// source family; a Bitmagnet feed and a website indexed by SearXNG might
	// share an original source, so these counts are not corroboration.
	appendRef := func(source, ref string) {
		if seen[ref] || len(r.Evidence) >= 16 {
			return
		}
		seen[ref] = true
		r.Evidence = append(r.Evidence, SweepDiscoveryEvidence{
			Source: source, EvidenceRef: ref, AncestryRef: sweepOpaqueRef("unqualified-source-family", source),
		})
	}
	if r.Bitmagnet != nil {
		for _, item := range r.Bitmagnet.Results {
			appendRef("bitmagnet", sweepOpaqueRef("bitmagnet", item.InfoHash, item.Title))
		}
	}
	for _, q := range r.External {
		for _, hit := range q.Hits {
			appendRef("searxng", sweepOpaqueRef("searxng", hit.URL, hit.Title, q.Language))
		}
	}
}
func runSweep(ctx context.Context, req SweepRequest) (SweepResult, error) {
	plan, e := makeSweepPlan(req)
	if e != nil {
		return SweepResult{}, e
	}
	result := SweepResult{State: "complete", DeadlineSeconds: int(sweepExecutionDeadline / time.Second), Plan: plan, External: []SweepExternalObservation{}, Failures: []string{}, Evidence: []SweepDiscoveryEvidence{}, ResearchProposals: []SweepDiscoveryDraft{}, CPEWrite: false,
		Warning: "Research/metadata retrieval only. External results are independent neutral observations; enrichment requires CPE #202 review, grounding, counterexamples and promotion authority."}
	if len(plan.Bitmagnet) > 0 {
		q := Request{Query: plan.Bitmagnet[0].Term, Filters: req.Filters, Expansions: []Expansion{}}
		for _, v := range plan.Bitmagnet[1:] {
			q.Expansions = append(q.Expansions, Expansion{Term: v.Term, Language: v.Language, ConceptID: v.ConceptID, Provenance: v.Provenance})
		}
		data, e := execute(ctx, q)
		if e != nil {
			result.Failures = append(result.Failures, "bitmagnet: "+e.Error())
		} else {
			result.Bitmagnet = &data
			if data.State != "complete" {
				result.Failures = append(result.Failures, "bitmagnet: "+data.State)
			}
		}
	}
	// Sequential external execution avoids burst patterns even when the local
	// backend permits parallel searches. The source-wide pacer independently
	// enforces inter-request spacing and cooldown on 403/429/503.
	knownTerms := map[string]bool{}
	for _, q := range plan.Bitmagnet {
		knownTerms[strings.ToLower(strings.TrimSpace(q.Term))] = true
	}
	for _, q := range plan.Searxng {
		knownTerms[strings.ToLower(strings.TrimSpace(q.Term))] = true
	}
	suggested := map[string]bool{}
	for _, item := range plan.Searxng {
		if ctx.Err() != nil {
			result.Failures = append(result.Failures, "external sweep canceled")
			break
		}
		packet, e := searchSearXNGDetailed(ctx, item.Term, item.Language)
		if e != nil {
			result.Failures = append(result.Failures, "searxng "+item.Language+": "+e.Error())
			// Rate/permission refusal stops the whole source, not just this lane.
			if errors.Is(e, errSearXNGBackoff) {
				break
			}
			continue
		}
		result.External = append(result.External, SweepExternalObservation{Term: item.Term, Language: item.Language, Hits: packet.Hits, Suggestions: packet.Suggestions})
		// Quarantined query-suggestion hypotheses: no second provider request,
		// no CPE authority write, no inference that a suggestion is an ontology fact.
		if item.Language != "und" {
			for _, hint := range packet.Suggestions {
				key := strings.ToLower(hint) + "\x00" + item.Language
				if knownTerms[strings.ToLower(strings.TrimSpace(hint))] || suggested[key] || len(result.ResearchProposals) >= 4 {
					continue
				}
				suggested[key] = true
				ref := sweepOpaqueRef("searxng-query-suggestion", item.Term, hint, item.Language)
				proposal, e := previewSweepDiscovery(SweepDiscoveryRequest{
					Kind: "lexicalization_candidate", CandidateLabel: hint, Language: item.Language,
					ConceptID: item.ConceptID, SnapshotDigest: plan.ProjectionDigest,
					Evidence: []SweepDiscoveryEvidence{{
						Source: "searxng", EvidenceRef: ref,
						AncestryRef: sweepOpaqueRef("unqualified-source-family", "searxng"),
					}},
				})
				if e == nil {
					result.ResearchProposals = append(result.ResearchProposals, proposal)
				}
			}
		}
	}
	addSweepEvidence(&result)
	if len(result.Failures) > 0 {
		result.State = "partial"
		if result.Bitmagnet == nil && len(result.External) == 0 {
			result.State = "failed"
		}
	}
	return result, nil
}
func registerSweeps(mux *http.ServeMux) {
	route := func(w http.ResponseWriter, r *http.Request, execute bool) {
		r.Body = http.MaxBytesReader(w, r.Body, contracts.SweepJSONBytes)
		defer r.Body.Close()
		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields()
		var request SweepRequest
		if e := d.Decode(&request); e != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid sweep request"})
			return
		}
		if e := d.Decode(new(any)); e != io.EOF {
			writeJSON(w, 400, map[string]string{"error": "trailing sweep request"})
			return
		}
		if !execute {
			plan, e := makeSweepPlan(request)
			if e != nil {
				writeJSON(w, 422, map[string]string{"error": e.Error()})
				return
			}
			writeJSON(w, 200, plan)
			return
		}
		select {
		case activeSweepSlots <- struct{}{}:
			defer func() { <-activeSweepSlots }()
		default:
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "sweep capacity is busy; no work started and no automatic retry was made"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), sweepExecutionDeadline)
		defer cancel()
		result, e := runSweep(ctx, request)
		if e != nil {
			writeJSON(w, 422, map[string]string{"error": e.Error()})
			return
		}
		writeJSON(w, 200, result)
	}
	mux.HandleFunc("POST /api/v1/sweeps/plan", func(w http.ResponseWriter, r *http.Request) { route(w, r, false) })
	mux.HandleFunc("POST /api/v1/sweeps/execute", func(w http.ResponseWriter, r *http.Request) { route(w, r, true) })
}
