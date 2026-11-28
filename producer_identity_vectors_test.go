package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

// These deliberately synthetic vectors are shared with the fixture consumer.
// Exercise the real producer so other language implementations can verify
// exact research-v1 identity bytes without redefining the producer contract.
func TestResearchProposalIdentitySharedSyntheticVectors(t *testing.T) {
	raw, err := os.ReadFile("testdata/producer-identity-synthetic-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		SchemaVersion string `json:"schema_version"`
		Vectors       []struct {
			Name          string                `json:"name"`
			Request       SweepDiscoveryRequest `json:"request"`
			CanonicalUTF8 string                `json:"canonical_utf8"`
			ProposalID    string                `json:"proposal_id"`
		} `json:"vectors"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.SchemaVersion != "bce-proposal-identity-synthetic-v1" || len(fixture.Vectors) != 14 {
		t.Fatal("unexpected shared synthetic identity fixture")
	}
	for _, vector := range fixture.Vectors {
		t.Run(vector.Name, func(t *testing.T) {
			draft, err := previewSweepDiscovery(vector.Request)
			if err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256([]byte(vector.CanonicalUTF8))
			if vector.ProposalID != "bce-sweep-v1-"+hex.EncodeToString(digest[:]) || draft.ProposalID != vector.ProposalID {
				t.Fatalf("producer identity differs from shared canonical bytes: got=%s expected=%s", draft.ProposalID, vector.ProposalID)
			}
			rawDraft, err := json.Marshal(draft)
			if err != nil {
				t.Fatal(err)
			}
			var envelope map[string]json.RawMessage
			if err := json.Unmarshal(rawDraft, &envelope); err != nil {
				t.Fatal(err)
			}
			if _, present := envelope["concept_id"]; present != (vector.Request.ConceptID != "") {
				t.Fatal("unmapped target omission changed without an identity migration")
			}
		})
	}
}
