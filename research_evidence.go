package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// ResearchEvidencePack is a separate public-source RESEARCH interchange. It
// cannot represent accepted CPE authority or a rights/redistribution decision.
type ResearchEvidencePack struct {
	SchemaVersion   string           `json:"schema_version"`
	PackID          string           `json:"pack_id"`
	AuthorityStatus string           `json:"authority_status"`
	ObservedOn      string           `json:"observed_on"`
	Scope           string           `json:"scope"`
	RightsSemantics string           `json:"rights_semantics"`
	Works           []ResearchWork   `json:"works"`
	Entities        []ResearchEntity `json:"entities"`
	Claims          []ResearchClaim  `json:"claims"`
	ArtCandidates   []ResearchArt    `json:"art_candidates"`
	SourceInventory []ResearchSource `json:"source_inventory"`
}
type ResearchWork struct {
	ID                           string `json:"id"`
	Title                        string `json:"title"`
	CreatorName                  string `json:"creator_name"`
	SourceRecordID               string `json:"source_record_id"`
	SourceURL                    string `json:"source_url"`
	Language                     string `json:"language"`
	FirstPublicationYearReported int    `json:"first_publication_year_reported"`
	FirstPublicationYearEvidence string `json:"first_publication_year_evidence"`
	Status                       string `json:"status"`
}
type ResearchEntity struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	Name string `json:"name"`
}
type ResearchClaim struct {
	ID           string `json:"id"`
	Subject      string `json:"subject"`
	Predicate    string `json:"predicate"`
	Object       string `json:"object"`
	SourceURL    string `json:"source_url"`
	SourceFamily string `json:"source_family"`
	EvidenceRole string `json:"evidence_role"`
	Status       string `json:"status"`
}
type ResearchRightsObservation struct {
	Claimant  string `json:"claimant"`
	Scope     string `json:"scope"`
	Statement string `json:"statement"`
	Status    string `json:"status"`
}
type ResearchArt struct {
	ID                string                    `json:"id"`
	RelatedWork       string                    `json:"related_work"`
	Title             string                    `json:"title"`
	Creator           string                    `json:"creator"`
	SourceURL         string                    `json:"source_url"`
	SourceFamily      string                    `json:"source_family"`
	SourceDescription string                    `json:"source_description"`
	RightsObservation ResearchRightsObservation `json:"rights_observation"`
	FileRetained      bool                      `json:"file_retained"`
	Downloaded        bool                      `json:"downloaded"`
}
type ResearchSource struct {
	ID              string `json:"id"`
	Purpose         string `json:"purpose"`
	URL             string `json:"url"`
	PreferredMethod string `json:"preferred_method"`
	Constraints     string `json:"constraints"`
	Role            string `json:"role"`
}

func researchURL(s string) bool {
	u, e := url.Parse(s)
	return e == nil && u.Scheme == "https" && u.Host != "" && u.User == nil
}

func validateResearchEvidence(p ResearchEvidencePack) error {
	if p.SchemaVersion != "public-lore-evidence-v0" ||
		p.AuthorityStatus != "research-only-no-accepted-cpe-facts" ||
		p.PackID == "" || len(p.PackID) > 128 ||
		p.Scope == "" || p.RightsSemantics == "" ||
		len(p.Works) == 0 || len(p.Works) > 128 ||
		len(p.Entities) > 256 || len(p.Claims) > 512 ||
		len(p.ArtCandidates) > 128 || len(p.SourceInventory) > 128 {
		return errors.New("invalid or non-research evidence packet")
	}
	ids := make(map[string]bool)
	add := func(id string) bool {
		if id == "" || len(id) > 160 || ids[id] {
			return false
		}
		ids[id] = true
		return true
	}
	works := make(map[string]bool)
	entities := make(map[string]bool)
	for _, w := range p.Works {
		if !add(w.ID) || !strings.HasPrefix(w.ID, "work:") ||
			!researchURL(w.SourceURL) || w.SourceRecordID == "" ||
			w.Title == "" || w.Language == "" ||
			w.Status != "public_research_candidate" {
			return errors.New("invalid or promoted research work")
		}
		works[w.ID] = true
	}
	for _, e := range p.Entities {
		if !add(e.ID) || e.Name == "" || e.Kind == "" {
			return errors.New("invalid research entity")
		}
		entities[e.ID] = true
	}
	for _, c := range p.Claims {
		if !add(c.ID) || !works[c.Subject] ||
			!researchURL(c.SourceURL) || c.Predicate == "" ||
			c.Object == "" || c.SourceFamily == "" || c.EvidenceRole == "" ||
			(c.Status != "source_report_not_independently_qualified" &&
			 c.Status != "research_relationship_candidate") {
			return errors.New("invalid or promoted source claim")
		}
		if strings.HasPrefix(c.Object, "fictional:") && !entities[c.Object] {
			return errors.New("orphaned research claim object")
		}
	}
	allowedRights := map[string]bool{
		"recorded_not_redistribution_approved": true,
		"item_level_status_unverified": true,
		"recorded_source_claim_not_specific_derivative_clearance": true,
	}
	for _, a := range p.ArtCandidates {
		if !add(a.ID) || !works[a.RelatedWork] ||
			!researchURL(a.SourceURL) || a.Title == "" ||
			a.SourceFamily == "" || a.Downloaded || a.FileRetained ||
			a.RightsObservation.Claimant == "" ||
			a.RightsObservation.Scope == "" ||
			a.RightsObservation.Statement == "" ||
			!allowedRights[a.RightsObservation.Status] {
			return errors.New("invalid or unqualified art/rights research")
		}
	}
	for _, s := range p.SourceInventory {
		if !add(s.ID) || !researchURL(s.URL) || s.Role == "" ||
			s.PreferredMethod == "" || s.Constraints == "" {
			return errors.New("invalid research source reference")
		}
	}
	return nil
}

// No network fetch, discovery execution, media download, or accepted CPE write.
func readResearchEvidence() (ResearchEvidencePack, string, error) {
	path := strings.TrimSpace(os.Getenv("BCE_RESEARCH_EVIDENCE_FILE"))
	expected := strings.ToLower(strings.TrimSpace(os.Getenv("BCE_RESEARCH_EVIDENCE_SHA256")))
	if path == "" || len(expected) != 64 {
		return ResearchEvidencePack{}, "", errors.New("pinned public research evidence not configured")
	}
	if _, e := hex.DecodeString(expected); e != nil {
		return ResearchEvidencePack{}, "", errors.New("invalid public research evidence digest")
	}
	f, e := os.Open(path)
	if e != nil {
		return ResearchEvidencePack{}, "", errors.New("public research evidence unavailable")
	}
	defer f.Close()
	data, e := io.ReadAll(io.LimitReader(f, 256<<10+1))
	if e != nil || len(data) == 0 || len(data) > 256<<10 {
		return ResearchEvidencePack{}, "", errors.New("public research evidence too large or unreadable")
	}
	sum := sha256.Sum256(data)
	actual := hex.EncodeToString(sum[:])
	if actual != expected {
		return ResearchEvidencePack{}, "", errors.New("public research evidence digest mismatch")
	}
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	var p ResearchEvidencePack
	if e = dec.Decode(&p); e != nil {
		return ResearchEvidencePack{}, "", errors.New("invalid public research evidence packet")
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return ResearchEvidencePack{}, "", errors.New("trailing public research evidence data")
	}
	if e = validateResearchEvidence(p); e != nil {
		return ResearchEvidencePack{}, "", e
	}
	return p, "sha256:" + actual, nil
}

func evidenceMatches[T any](records []T, q string) []T {
	result := []T{}
	for _, item := range records {
		b, _ := json.Marshal(item)
		if q == "" || strings.Contains(strings.ToLower(string(b)), q) {
			result = append(result, item)
		}
	}
	return result
}

func registerResearchEvidence(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/research-evidence", func(w http.ResponseWriter, r *http.Request) {
		p, digest, err := readResearchEvidence()
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
			return
		}
		section := r.URL.Query().Get("section")
		q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
		if len([]rune(q)) > 256 || (q != "" && section == "") {
			writeJSON(w, 422, map[string]string{"error": "bounded query requires an explicit section"})
			return
		}
		envelope := map[string]any{
			"pack_id": p.PackID,
			"digest": digest,
			"authority_status": p.AuthorityStatus,
			"rights_semantics": p.RightsSemantics,
			"warnings": []string{"Source-reported research only; not accepted CPE facts or permission to redistribute art"},
		}
		switch section {
		case "":
			envelope["pack"] = p
		case "works":
			items := evidenceMatches(p.Works, q)
			envelope["items"], envelope["count"] = items, len(items)
		case "entities":
			items := evidenceMatches(p.Entities, q)
			envelope["items"], envelope["count"] = items, len(items)
		case "claims":
			items := evidenceMatches(p.Claims, q)
			envelope["items"], envelope["count"] = items, len(items)
		case "art_candidates":
			items := evidenceMatches(p.ArtCandidates, q)
			envelope["items"], envelope["count"] = items, len(items)
		case "source_inventory":
			items := evidenceMatches(p.SourceInventory, q)
			envelope["items"], envelope["count"] = items, len(items)
		default:
			writeJSON(w, 422, map[string]string{"error": "unknown public research evidence section"})
			return
		}
		writeJSON(w, 200, envelope)
	})
}
