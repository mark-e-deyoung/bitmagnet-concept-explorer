package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/mark-e-deyoung/bitmagnet-concept-explorer/internal/contracts"
	"io"
	"net/http"
	"strings"
)

// DiscoveryProposal is a local, provisional export envelope. It NEVER updates CPE.
type DiscoveryProposal struct {
	SchemaVersion  string   `json:"schema_version"`
	ProposalID     string   `json:"proposal_id"`
	Producer       string   `json:"producer"`
	Kind           string   `json:"proposal_kind"`
	CandidateLabel string   `json:"candidate_label"`
	ConceptID      *string  `json:"concept_id"`
	SourceScope    string   `json:"source_scope"`
	Evidence       []string `json:"evidence_refs"`
	Confidence     string   `json:"confidence"`
	Uncertainty    string   `json:"uncertainty"`
	Privacy        string   `json:"privacy"`
	State          string   `json:"state"`
}

func makeProposal(kind, label string, refs []string) (DiscoveryProposal, error) {
	kind = strings.TrimSpace(kind)
	label = strings.TrimSpace(label)
	allowed := map[string]bool{"lexicalization_candidate": true, "hard_negative_or_collision": true, "coverage_gap": true, "source_taxonomy_drift": true, "unknown_residual_cluster": true}
	if !allowed[kind] || label == "" || len([]rune(label)) > 256 || len(refs) == 0 || len(refs) > 20 {
		return DiscoveryProposal{}, errors.New("invalid discovery proposal")
	}
	clean := make([]string, 0, len(refs))
	for _, r := range refs {
		r = strings.TrimSpace(r)
		if !strings.HasPrefix(r, "sha256:") || len(r) != 71 {
			return DiscoveryProposal{}, errors.New("evidence refs must be sha256 digests")
		}
		for _, c := range r[7:] {
			if !strings.ContainsRune("0123456789abcdef", c) {
				return DiscoveryProposal{}, errors.New("invalid evidence digest")
			}
		}
		clean = append(clean, r)
	}
	// Retain caller order as part of proposal identity until CPE agreement on canonicalization.
	canonical, _ := json.Marshal(struct {
		K, L string
		R    []string
	}{kind, label, clean})
	hash := sha256.Sum256(canonical)
	return DiscoveryProposal{SchemaVersion: "research-v0", ProposalID: "bce-" + hex.EncodeToString(hash[:]), Producer: "bitmagnet-concept-explorer", Kind: kind, CandidateLabel: label, ConceptID: nil, SourceScope: "bitmagnet-metadata", Evidence: clean, Confidence: "unassessed", Uncertainty: "Retrieval observation only; independent semantic assessment required", Privacy: "evidence digests only; label requires local review before sharing", State: "local-export-only"}, nil
}
func registerProposals(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/discovery-proposals/preview", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, contracts.LegacyDiscoveryJSONBytes)
		defer r.Body.Close()
		var input struct {
			Kind     string   `json:"proposal_kind"`
			Label    string   `json:"candidate_label"`
			Evidence []string `json:"evidence_refs"`
		}
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&input); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid request"})
			return
		}
		var extra any
		if err := dec.Decode(&extra); err != io.EOF {
			writeJSON(w, 400, map[string]string{"error": "trailing JSON"})
			return
		}
		proposal, err := makeProposal(input.Kind, input.Label, input.Evidence)
		if err != nil {
			writeJSON(w, 422, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, proposal)
	})
}
