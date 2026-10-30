package main

import (
	"encoding/json"
	"errors"
	"github.com/mark-e-deyoung/bitmagnet-concept-explorer/internal/contracts"
	"io"
	"net/http"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const observedGraphRecordLimit = 200
const observedGraphEdgeLimit = 512

// Observed graph is a projection of a *supplied* execution receipt, never an
// ontology search or a database-wide concept graph. Replaying an untrusted
// input here does not attest that the underlying results were retrieved.
type ObservedGraphRequest struct {
	Results []Observation `json:"results"`
}
type ObservedConceptEdge struct {
	InfoHash           string `json:"info_hash"`
	ConceptID          string `json:"concept_id"`
	Surface            string `json:"surface"`
	MatchedTerm        string `json:"matched_term"`
	FormLanguage       string `json:"form_language,omitempty"`
	AssessmentEligible bool   `json:"assessment_eligible"`
	EvidenceRole       string `json:"evidence_role"`
}
type ObservedConceptCount struct {
	ID                       string `json:"id"`
	Label                    string `json:"label"`
	DistinctReturnedTorrents int    `json:"distinct_returned_torrents"`
}
type ObservedGraph struct {
	Version          string                 `json:"version"`
	Scope            string                 `json:"scope"`
	ProjectionDigest string                 `json:"projection_digest"`
	AuthorityStatus  string                 `json:"authority_status"`
	Status           string                 `json:"status"`
	Truncated        bool                   `json:"truncated"`
	ReturnedRecords  int                    `json:"returned_records"`
	Concepts         []ObservedConceptCount `json:"concepts"`
	EvidenceEdges    []ObservedConceptEdge  `json:"evidence_edges"`
	Warnings         []string               `json:"warnings"`
}

func isWordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r) }
func boundedLexicalHit(title, term string) bool {
	if term == "" || title == "" {
		return false
	}
	// This is an approximate, Unicode case-folded *retrieval* heuristic,
	// not word segmentation or a language-specific semantic match.
	a, b := []rune(strings.ToLower(title)), []rune(strings.ToLower(term))
	if len(b) == 0 || len(b) > len(a) {
		return false
	}
	for i := 0; i <= len(a)-len(b); i++ {
		same := true
		for j := range b {
			if a[i+j] != b[j] {
				same = false
				break
			}
		}
		if !same {
			continue
		}
		if i > 0 && isWordRune(a[i-1]) {
			continue
		}
		if i+len(b) < len(a) && isWordRune(a[i+len(b)]) {
			continue
		}
		return true
	}
	return false
}
func buildObservedGraph(catalog ConceptCatalog, records []Observation) (ObservedGraph, error) {
	if len(records) > observedGraphRecordLimit {
		return ObservedGraph{}, errors.New("too many supplied result records")
	}
	out := ObservedGraph{
		Version: "observed-lexical-v0", Scope: "supplied_bounded_bitmagnet_result_metadata_not_corpus_global",
		ProjectionDigest: catalog.Digest, AuthorityStatus: catalog.AuthorityStatus,
		Status: "no_lexical_evidence", ReturnedRecords: len(records),
		Concepts: []ObservedConceptCount{}, EvidenceEdges: []ObservedConceptEdge{},
		Warnings: []string{"Graph input is caller-supplied and not an authenticated Bitmagnet retrieval receipt", "Only title lexical matches are implemented; these are candidates, never accepted CPE assertions or proof of depiction/genre/age suitability", "CPE preview projection is not an accepted semantic authority; no match is not concept absence"},
	}
	if len(records) == 0 {
		out.Status = "no_returned_records"
		return out, nil
	}
	type form struct{ term, language string }
	forms := map[string][]form{}
	labels := map[string]string{}
	for _, node := range catalog.Concepts {
		labels[node.ID] = node.Label
		forms[node.ID] = append(forms[node.ID], form{node.Label, ""})
		for _, l := range node.Lexicalizations {
			forms[node.ID] = append(forms[node.ID], form{l.Term, l.Language})
		}
	}
	ids := make([]string, 0, len(forms))
	for id := range forms {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	counts := map[string]map[string]bool{}
	seenEdge := map[string]bool{}
	for _, rec := range records {
		if rec.InfoHash == "" || utf8.RuneCountInString(rec.Title) > 1024 {
			return ObservedGraph{}, errors.New("invalid supplied bounded result")
		}
		for _, id := range ids {
			for _, f := range forms[id] {
				if !boundedLexicalHit(rec.Title, f.term) {
					continue
				}
				key := rec.InfoHash + "\x00" + id
				if counts[id] == nil {
					counts[id] = map[string]bool{}
				}
				counts[id][rec.InfoHash] = true
				// One trace per resource-concept pair; provenance of multiple lexical
				// forms remains an explicit future evidence detail, never a false count.
				if seenEdge[key] {
					continue
				}
				seenEdge[key] = true
				if len(out.EvidenceEdges) >= observedGraphEdgeLimit {
					out.Truncated = true
					continue
				}
				out.EvidenceEdges = append(out.EvidenceEdges, ObservedConceptEdge{
					InfoHash: rec.InfoHash, ConceptID: id, Surface: "title", MatchedTerm: f.term, FormLanguage: f.language,
					AssessmentEligible: false, EvidenceRole: "candidate_lexical_match",
				})
			}
		}
	}
	for _, id := range ids {
		if n := len(counts[id]); n > 0 {
			out.Concepts = append(out.Concepts, ObservedConceptCount{ID: id, Label: labels[id], DistinctReturnedTorrents: n})
		}
	}
	if len(out.Concepts) > 0 {
		out.Status = "lexical_candidates_observed"
	}
	if out.Truncated {
		out.Warnings = append(out.Warnings, "Evidence edge output cap reached; graph is incomplete and counts must not be treated as exhaustively explained")
	}
	return out, nil
}
func registerObservedGraph(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/query/observed-graph", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, contracts.ObservedGraphJSONBytes)
		defer r.Body.Close()
		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields()
		var input ObservedGraphRequest
		if e := d.Decode(&input); e != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid bounded receipt input"})
			return
		}
		if e := d.Decode(new(any)); e != io.EOF {
			writeJSON(w, 400, map[string]string{"error": "trailing receipt input"})
			return
		}
		catalog, e := readConceptCatalog()
		if e != nil {
			writeJSON(w, 503, map[string]string{"error": e.Error()})
			return
		}
		out, e := buildObservedGraph(catalog, input.Results)
		if e != nil {
			writeJSON(w, 422, map[string]string{"error": e.Error()})
			return
		}
		writeJSON(w, 200, out)
	})
}
