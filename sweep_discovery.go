package main

import (
	"crypto/sha256"
	"encoding/hex"
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

// Research proposals are durable in their *identifier*, but never persisted
// by this service or submitted to CPE. The CPE #202 adjudication process owns
// whether a candidate is mapped, accepted, split, rejected or deferred.
type SweepDiscoveryEvidence struct {
	Source      string `json:"source"`
	EvidenceRef string `json:"evidence_ref"`
	AncestryRef string `json:"ancestry_ref"`
}
type SweepDiscoveryRequest struct {
	Kind           string                   `json:"proposal_kind"`
	CandidateLabel string                   `json:"candidate_label"`
	Language       string                   `json:"language"`
	ConceptID      string                   `json:"concept_id,omitempty"`
	SnapshotDigest string                   `json:"projection_digest,omitempty"`
	Evidence       []SweepDiscoveryEvidence `json:"evidence"`
}
type SweepDiscoveryDraft struct {
	SchemaVersion         string                   `json:"schema_version"`
	ProposalID            string                   `json:"proposal_id"`
	Kind                  string                   `json:"proposal_kind"`
	CandidateLabel        string                   `json:"candidate_label"`
	Language              string                   `json:"language"`
	ConceptID             string                   `json:"concept_id,omitempty"`
	ProjectionDigest      string                   `json:"projection_digest,omitempty"`
	Evidence              []SweepDiscoveryEvidence `json:"evidence"`
	DistinctAncestryCount int                      `json:"distinct_ancestry_count"`
	State                 string                   `json:"state"`
	Authority             string                   `json:"authority"`
	Warnings              []string                 `json:"warnings"`
}

var cpeDiscoveryKinds = map[string]bool{
	"lexicalization_candidate": true, "source_mapping_candidate": true, "concept_candidate": true,
	"concept_split_candidate": true, "concept_merge_or_alias_candidate": true, "relationship_candidate": true,
	"hard_negative_or_collision": true, "definition_or_boundary_drift": true,
	"coverage_gap": true, "source_taxonomy_drift": true, "unknown_residual_cluster": true,
}

func shaEvidenceRef(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, b := range value[7:] {
		if !strings.ContainsRune("0123456789abcdef", b) {
			return false
		}
	}
	return true
}
func previewSweepDiscovery(input SweepDiscoveryRequest) (SweepDiscoveryDraft, error) {
	if !cpeDiscoveryKinds[input.Kind] || !utf8.ValidString(input.CandidateLabel) || len([]rune(strings.TrimSpace(input.CandidateLabel))) < 2 || len([]rune(input.CandidateLabel)) > 128 ||
		!sweepLanguage(input.Language) || len(input.Evidence) < 1 || len(input.Evidence) > 16 ||
		len(input.ConceptID) > 128 {
		return SweepDiscoveryDraft{}, errors.New("invalid concept discovery candidate envelope")
	}
	for _, r := range input.CandidateLabel {
		if unicode.IsControl(r) {
			return SweepDiscoveryDraft{}, errors.New("control character in discovery label")
		}
	}
	if input.SnapshotDigest != "" && !shaEvidenceRef(input.SnapshotDigest) {
		return SweepDiscoveryDraft{}, errors.New("invalid CPE research projection digest")
	}
	seen := map[string]string{}
	ancestries := map[string]bool{}
	clean := []SweepDiscoveryEvidence{}
	for _, v := range input.Evidence {
		if v.Source != "bitmagnet" && v.Source != "searxng" && v.Source != "sovereign_model" {
			return SweepDiscoveryDraft{}, errors.New("unsupported proposal evidence source")
		}
		if !shaEvidenceRef(v.EvidenceRef) || !shaEvidenceRef(v.AncestryRef) {
			return SweepDiscoveryDraft{}, errors.New("raw source content or invalid ancestry ref rejected")
		}
		key := v.Source + "\x00" + v.EvidenceRef
		if priorAncestry, ok := seen[key]; ok {
			if priorAncestry != v.AncestryRef {
				return SweepDiscoveryDraft{}, errors.New("conflicting ancestry for duplicate source evidence")
			}
			continue
		}
		seen[key] = v.AncestryRef
		ancestries[v.AncestryRef] = true
		clean = append(clean, v)
	}
	if len(clean) == 0 {
		return SweepDiscoveryDraft{}, errors.New("proposal has no distinct evidence")
	}
	sort.Slice(clean, func(i, j int) bool {
		if clean[i].Source != clean[j].Source {
			return clean[i].Source < clean[j].Source
		}
		return clean[i].EvidenceRef < clean[j].EvidenceRef
	})
	// Research-v1 identity includes the supplied snapshot reference, language,
	// concept target and normalized source ancestry. Exact duplicate evidence is
	// idempotent; conflicting ancestry for one source/evidence ref is rejected
	// above rather than resolved by input order.
	canonical, _ := json.Marshal(struct {
		Version    string                   `json:"version"`
		Kind       string                   `json:"kind"`
		Label      string                   `json:"label"`
		Language   string                   `json:"language"`
		ConceptID  string                   `json:"concept_id"`
		Projection string                   `json:"projection"`
		Evidence   []SweepDiscoveryEvidence `json:"evidence"`
	}{"cpe-discovery-proposal-research-v1", input.Kind, strings.TrimSpace(input.CandidateLabel), strings.ToLower(input.Language), input.ConceptID, input.SnapshotDigest, clean})
	digest := sha256.Sum256(canonical)
	warnings := []string{"Digest-only evidence does not verify that a source observation is authentic or semantically correct", "Duplicated search engines or same-origin pages do not add independent corroboration", "Local research preview only: CPE #202 independent review and explicit CPE authority are required for any update"}
	return SweepDiscoveryDraft{
		SchemaVersion: "cpe-discovery-proposal-research-v1", ProposalID: "bce-sweep-v1-" + hex.EncodeToString(digest[:]),
		Kind: input.Kind, CandidateLabel: strings.TrimSpace(input.CandidateLabel),
		Language: strings.ToLower(input.Language), ConceptID: input.ConceptID,
		ProjectionDigest: input.SnapshotDigest, Evidence: clean, DistinctAncestryCount: len(ancestries),
		State: "local-review-export-only", Authority: "not_accepted_not_transmitted",
		Warnings: warnings,
	}, nil
}
func registerSweepDiscovery(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/sweeps/discovery/preview", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, contracts.SweepDiscoveryJSONBytes)
		defer r.Body.Close()
		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields()
		var input SweepDiscoveryRequest
		if e := d.Decode(&input); e != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid research proposal input"})
			return
		}
		if e := d.Decode(new(any)); e != io.EOF {
			writeJSON(w, 400, map[string]string{"error": "trailing research proposal data"})
			return
		}
		out, e := previewSweepDiscovery(input)
		if e != nil {
			writeJSON(w, 422, map[string]string{"error": e.Error()})
			return
		}
		writeJSON(w, 200, out)
	})
}
