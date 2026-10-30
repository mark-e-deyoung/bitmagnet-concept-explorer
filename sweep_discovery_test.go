package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func digestFixture(ch string) string { return "sha256:" + strings.Repeat(ch, 64) }
func TestResearchProposalDeterministicAcrossSourceOrder(t *testing.T) {
	a := SweepDiscoveryRequest{Kind: "lexicalization_candidate", CandidateLabel: "แอนิเมชัน", Language: "th",
		ConceptID: "media.production.animation", SnapshotDigest: digestFixture("a"),
		Evidence: []SweepDiscoveryEvidence{
			{Source: "searxng", EvidenceRef: digestFixture("b"), AncestryRef: digestFixture("c")},
			{Source: "bitmagnet", EvidenceRef: digestFixture("d"), AncestryRef: digestFixture("c")}, // same ancestry
		},
	}
	out, e := previewSweepDiscovery(a)
	if e != nil {
		t.Fatal(e)
	}
	if out.DistinctAncestryCount != 1 || out.State != "local-review-export-only" || out.Authority != "not_accepted_not_transmitted" {
		t.Fatalf("false independence or authority %#v", out)
	}
	b := a
	b.Evidence = []SweepDiscoveryEvidence{a.Evidence[1], a.Evidence[0], a.Evidence[0]}
	two, e := previewSweepDiscovery(b)
	if e != nil {
		t.Fatal(e)
	}
	if out.ProposalID != two.ProposalID || len(two.Evidence) != 2 {
		t.Fatalf("unstable ancestry canonicalization %#v %#v", out, two)
	}
	if len(out.Evidence) != 2 {
		t.Fatal("unexpected removal of cross-consumer provenance")
	}
}
func TestResearchProposalRejectsRawPrivateMetadataAndUntrustedSource(t *testing.T) {
	original := SweepDiscoveryRequest{Kind: "relationship_candidate", CandidateLabel: "Example distinction", Language: "en", Evidence: []SweepDiscoveryEvidence{{Source: "searxng", EvidenceRef: digestFixture("a"), AncestryRef: digestFixture("b")}}}
	for _, alter := range []func(*SweepDiscoveryRequest){
		func(r *SweepDiscoveryRequest) { r.Evidence[0].EvidenceRef = "raw private torrent title" },
		func(r *SweepDiscoveryRequest) { r.Evidence[0].Source = "public-cloud-model" },
		func(r *SweepDiscoveryRequest) { r.Language = "../../secret" },
		func(r *SweepDiscoveryRequest) { r.Kind = "automatic_accept" },
	} {
		r := original
		r.Evidence = append([]SweepDiscoveryEvidence(nil), original.Evidence...)
		alter(&r)
		if _, e := previewSweepDiscovery(r); e == nil {
			t.Errorf("accepted invalid private or unauthorized proposal %#v", r)
		}
	}
}
func TestResearchProposalPreviewAPINeverCommitsToCPE(t *testing.T) {
	input := SweepDiscoveryRequest{Kind: "hard_negative_or_collision", CandidateLabel: "ROM false positive", Language: "en",
		Evidence: []SweepDiscoveryEvidence{{Source: "searxng", EvidenceRef: digestFixture("d"), AncestryRef: digestFixture("e")}}}
	body, e := json.Marshal(input)
	if e != nil {
		t.Fatal(e)
	}
	w := httptest.NewRecorder()
	handler().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/sweeps/discovery/preview", bytes.NewReader(body)))
	if w.Code != 200 {
		t.Fatalf("unexpected status: %d %s", w.Code, w.Body.String())
	}
	var out SweepDiscoveryDraft
	if e := json.Unmarshal(w.Body.Bytes(), &out); e != nil || out.State != "local-review-export-only" || out.ProposalID == "" {
		t.Fatalf("invalid research envelope %v %#v", e, out)
	}
	if strings.Contains(w.Body.String(), `"cpe_write":true`) {
		t.Fatal("proposal falsely granted CPE authority")
	}
}
