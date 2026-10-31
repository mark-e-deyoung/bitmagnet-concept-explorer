package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Lexical research must never turn candidate translations into accepted
// semantic equivalence without source-native sense evidence and adjudication.
func validatePilotLexemes(raw []byte, graph ConceptSnapshot) error {
	var packet struct {
		SchemaVersion string `json:"schema_version"`
		PackID string `json:"pack_id"`
		SourceGraphSnapshotID string `json:"source_graph_snapshot_id"`
		AuthorityStatus string `json:"authority_status"`
		ProvenanceNote string `json:"provenance_note"`
		QualificationRule string `json:"qualification_rule"`
		Terms []struct {
			ConceptID string `json:"concept_id"`
			Language string `json:"language"`
			Term string `json:"term"`
			CandidateRelation string `json:"candidate_relation"`
			SourceFamily string `json:"source_family"`
			LexicalSenseRef *string `json:"lexical_sense_ref"`
			NativeLanguageEvidenceRefs []string `json:"native_language_evidence_refs"`
			TranslationRelationEvidenceRefs []string `json:"translation_relation_evidence_refs"`
			IndependentReviewStatus string `json:"independent_review_status"`
			AllowedAsAcceptedCPE bool `json:"allowed_as_accepted_cpe"`
		} `json:"terms"`
	}
	if err := json.Unmarshal(raw, &packet); err != nil {
		return err
	}
	if packet.SchemaVersion != "bce-lexical-evidence-v0" ||
		packet.SourceGraphSnapshotID != graph.SnapshotID ||
		packet.AuthorityStatus != "research-only-no-accepted-cpe-lexicalizations" ||
		packet.ProvenanceNote == "" || packet.QualificationRule == "" {
		return fmt.Errorf("invalid lexical candidate authority or graph identity")
	}
	known := make(map[string]bool)
	for _, c := range graph.Concepts {
		for _, l := range c.Lexicalizations {
			known[c.ID+"\x00"+l.Language+"\x00"+l.Term] = true
		}
	}
	seen := make(map[string]bool)
	for _, item := range packet.Terms {
		key := item.ConceptID+"\x00"+item.Language+"\x00"+item.Term
		if !known[key] || seen[key] ||
			item.CandidateRelation != "unqualified_label_candidate" ||
			item.SourceFamily != "pilot-author-provided-unverified" ||
			item.LexicalSenseRef != nil ||
			len(item.NativeLanguageEvidenceRefs) != 0 ||
			len(item.TranslationRelationEvidenceRefs) != 0 ||
			item.IndependentReviewStatus != "not_reviewed" ||
			item.AllowedAsAcceptedCPE {
			return fmt.Errorf("unqualified or falsely promoted lexicalization: %s", key)
		}
		seen[key] = true
	}
	if len(seen) != len(known) {
		return fmt.Errorf("lexical evidence coverage incomplete: got %d expected %d", len(seen), len(known))
	}
	return nil
}

func TestPublicGutenbergLexicalEvidenceNotPromoted(t *testing.T) {
	graphRaw, err := os.ReadFile(filepath.Join(publicPackDir, "graph-preview.json"))
	if err != nil {
		t.Fatal(err)
	}
	var graph ConceptSnapshot
	if err := json.Unmarshal(graphRaw, &graph); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(publicPackDir, "lexical-evidence.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := validatePilotLexemes(raw, graph); err != nil {
		t.Fatal(err)
	}
	mutations := []struct {
		original string
		injected string
	}{
		{`"allowed_as_accepted_cpe": false`, `"allowed_as_accepted_cpe": true`},
		{`"candidate_relation": "unqualified_label_candidate"`, `"candidate_relation": "exact_equivalence"`},
		{`"independent_review_status": "not_reviewed"`, `"independent_review_status": "approved"`},
		{`"source_family": "pilot-author-provided-unverified"`, `"source_family": "independent-native-source"`},
	}
	for _, mutation := range mutations {
		changed := []byte(stringReplaceOnce(string(raw), mutation.original, mutation.injected))
		if err := validatePilotLexemes(changed, graph); err == nil {
			t.Errorf("unearned lexical qualification accepted: %q", mutation.injected)
		}
	}
}

func stringReplaceOnce(s, old, replacement string) string {
	for i := 0; i+len(old) <= len(s); i++ {
		if s[i:i+len(old)] == old {
			return s[:i] + replacement + s[i+len(old):]
		}
	}
	return s
}
