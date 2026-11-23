package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

func fixtureExpected() Expected {
	return Expected{
		SourceLockSHA256:    strings.Repeat("a", 64),
		SnapshotBytesSHA256: strings.Repeat("b", 64),
		ProducerRevision:    strings.Repeat("c", 40),
		EvidenceSHA256:      strings.Repeat("d", 64),
		SourceSystem:        "fixture.public.lexical.source",
	}
}
func fixtureBody() map[string]any {
	e := fixtureExpected()
	return map[string]any{
		"schema_version": proposalVersion,
		"authority":      "RESEARCH_PROPOSAL_ONLY",
		"adjudication":   "UNADJUDICATED",
		"proposal_kind":  "lexicalization_candidate",
		"producer": map[string]any{
			"component":      "bce.independent_python_consumer.r35",
			"exact_revision": e.ProducerRevision,
		},
		"source_projection": map[string]any{
			"authority":             "research_only",
			"source_lock_sha256":    e.SourceLockSHA256,
			"snapshot_bytes_sha256": e.SnapshotBytesSHA256,
		},
		"observation": map[string]any{
			"target_concept_id": "public:literary-work",
			"language":          "th",
			"observation_type":  "native_source_term",
			"native_term":       "งานวรรณกรรม",
			"consumer_status":   "NO_LEXICALIZATION",
			"consumer_impact":   "RETRIEVAL_MISS_OR_MAPPING_GAP",
			"source_system":     e.SourceSystem,
		},
		"evidence": map[string]any{
			"provenance_digest":         e.EvidenceSHA256,
			"source_ancestry":           []any{e.EvidenceSHA256},
			"support_count":             json.Number("1"),
			"independent_corroboration": "NOT_ESTABLISHED",
			"confidence":                nil,
			"falsifier_status":          "NOT_YET_CHALLENGED",
		},
		"mapping": map[string]any{
			"candidate_relation":     "unmapped",
			"equivalence_status":     "UNKNOWN",
			"classification":         "NATIVE_SOURCE_TERM_NOT_IN_PINNED_LEXICAL_VIEW",
			"candidate_not_accepted": true,
		},
		"privacy": map[string]any{
			"class":                                "synthetic_public_only",
			"raw_private_content_included":         false,
			"sanitization_independently_qualified": false,
		},
	}
}
func encodeFixture(t *testing.T, body map[string]any) []byte {
	t.Helper()
	canon := new(bytes.Buffer)
	e := json.NewEncoder(canon)
	e.SetEscapeHTML(false)
	if err := e.Encode(body); err != nil {
		t.Fatal(err)
	}
	b := bytes.TrimSuffix(canon.Bytes(), []byte("\n"))
	sum := sha256.Sum256(append([]byte(proposalVersion+"\x00"), b...))
	data := map[string]any{}
	for k, v := range body {
		data[k] = v
	}
	data["proposal_id"] = "proposal:sha256:" + hex.EncodeToString(sum[:])
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func rejected(t *testing.T, raw []byte, expected string) {
	t.Helper()
	_, err := validate(raw, fixtureExpected())
	if err == nil || err.Error() != expected {
		t.Fatalf("expected %s, got %v", expected, err)
	}
}

func TestR37AdmitsOnlySyntheticUnadjudicatedCandidate(t *testing.T) {
	raw := encodeFixture(t, fixtureBody())
	got, err := validate(raw, fixtureExpected())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "VALID_RESEARCH_PROPOSAL_ONLY" ||
		got.Disposition != "DEFER_INDEPENDENT_GROUNDING" ||
		got.AcceptedSemanticChanges != 0 || got.IndependentCorroboration != "NOT_ESTABLISHED" {
		t.Fatalf("unqualified authority or corroboration: %+v", got)
	}
}
func TestR37ReproducibleProposalIdentity(t *testing.T) {
	a := encodeFixture(t, fixtureBody())
	b := encodeFixture(t, fixtureBody())
	if !bytes.Equal(a, b) {
		t.Fatal("same synthetic fixture not deterministic")
	}
}
func TestR37QualifiedSourceRevisionAndEvidenceRequired(t *testing.T) {
	tests := []struct{ field, expected string }{
		{"source", "SOURCE_PROJECTION_PIN_MISMATCH"},
		{"evidence", "UNQUALIFIED_EVIDENCE_ANCESTRY"},
		{"producer", "PRODUCER_IDENTITY_MISMATCH"},
	}
	for _, tc := range tests {
		t.Run(tc.field, func(t *testing.T) {
			b := fixtureBody()
			switch tc.field {
			case "source":
				b["source_projection"].(map[string]any)["source_lock_sha256"] = strings.Repeat("e", 64)
			case "evidence":
				b["evidence"].(map[string]any)["provenance_digest"] = strings.Repeat("e", 64)
			case "producer":
				b["producer"].(map[string]any)["exact_revision"] = strings.Repeat("e", 40)
			}
			rejected(t, encodeFixture(t, b), tc.expected)
		})
	}
}
func TestR37PromotedAuthorityNotLegitimizedByRehashedID(t *testing.T) {
	b := fixtureBody()
	b["authority"] = "ACCEPTED"
	rejected(t, encodeFixture(t, b), "UNAUTHORIZED_SEMANTIC_AUTHORITY")
}
func TestR37NoInventedIndependentSourceOrSupport(t *testing.T) {
	for _, item := range []string{"fake_independence", "duplicate_ancestry", "inflated_count"} {
		t.Run(item, func(t *testing.T) {
			b := fixtureBody()
			ev := b["evidence"].(map[string]any)
			switch item {
			case "fake_independence":
				ev["independent_corroboration"] = "VERIFIED"
			case "duplicate_ancestry":
				ev["source_ancestry"] = []any{fixtureExpected().EvidenceSHA256, fixtureExpected().EvidenceSHA256}
			case "inflated_count":
				ev["support_count"] = json.Number("2")
			}
			rejected(t, encodeFixture(t, b), "UNQUALIFIED_EVIDENCE_ANCESTRY")
		})
	}
}
func TestR37NoSyntheticFixturePrivacyUpgrade(t *testing.T) {
	b := fixtureBody()
	b["privacy"].(map[string]any)["sanitization_independently_qualified"] = true
	rejected(t, encodeFixture(t, b), "UNQUALIFIED_PRIVACY_CLAIM")
}
func TestR37NoAutomaticEquivalencePromotion(t *testing.T) {
	b := fixtureBody()
	b["mapping"].(map[string]any)["equivalence_status"] = "EXACT"
	rejected(t, encodeFixture(t, b), "UNAUTHORIZED_MAPPING_PROMOTION")
}
func TestR37BindsExactCandidateContent(t *testing.T) {
	original := encodeFixture(t, fixtureBody())
	tampered := bytes.Replace(original, []byte("NO_LEXICALIZATION"), []byte("OBSERVED"), 1)
	if bytes.Equal(original, tampered) {
		t.Fatal("failed to mutate fixture")
	}
	rejected(t, tampered, "PROPOSAL_IDENTITY_MISMATCH")
}
func TestR37DuplicatesJSONKeysFailClosed(t *testing.T) {
	rejected(t, []byte(`{"proposal_id":"x","proposal_id":"y"}`), "DUPLICATE_JSON_KEY")
	rejected(t, []byte(`{"nested":{"id":1,"\u0069d":2}}`), "DUPLICATE_JSON_KEY")
}
func TestR37InvalidUTF8AndOversizeFailClosed(t *testing.T) {
	rejected(t, []byte{0xff}, "INVALID_UTF8")
	rejected(t, []byte(strings.Repeat(" ", maxProposalBytes+1)), "OVERSIZED_PROPOSAL")
}
func TestR37KnownConsumerCapabilityGapRemainsDeferred(t *testing.T) {
	b := fixtureBody()
	b["proposal_kind"] = "coverage_gap"
	obs := b["observation"].(map[string]any)
	obs["observation_type"] = "consumer_lookup_miss"
	obs["native_term"] = nil
	obs["consumer_status"] = "OBSERVED"
	b["mapping"].(map[string]any)["classification"] = "KNOWN_CONCEPT_RETRIEVAL_MISS_NOT_ONTOLOGY_GAP"
	got, err := validate(encodeFixture(t, b), fixtureExpected())
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != "coverage_gap" || got.AcceptedSemanticChanges != 0 {
		t.Fatalf("consumer capability gap became accepted ontology: %+v", got)
	}
}
