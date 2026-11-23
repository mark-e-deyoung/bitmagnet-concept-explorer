package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func feedbackFixture(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/graphql" || r.Method != http.MethodPost {
			t.Errorf("unexpected metadata endpoint")
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("BCE_BITMAGNET_GRAPHQL", srv.URL+"/graphql")
	return srv
}

func feedbackInput() FeedbackPreviewRequest {
	return FeedbackPreviewRequest{
		Query:          Request{Query: "fictional animation"},
		ProposalKind:   "lexicalization_candidate",
		CandidateLabel: "งานแอนิเมชัน",
		Language:       "th",
	}
}

func TestFeedbackBindsActualBoundedMetadataWithPrivateStableAncestry(t *testing.T) {
	feedbackFixture(t, `{"data":{"torrentContent":{"search":{"hasNextPage":false,"items":[
	{"title":"PRIVATE FIXTURE Example Alpha","contentType":"movie","torrent":{"infoHash":"1111111111111111111111111111111111111111"}},
	{"title":"PRIVATE FIXTURE Example Beta","contentType":"movie","torrent":{"infoHash":"2222222222222222222222222222222222222222"}}
	]}}}}`)
	key := []byte(strings.Repeat("K", 32))
	a, e := buildFeedbackPreview(context.Background(), feedbackInput(), key)
	if e != nil {
		t.Fatal(e)
	}
	b, e := buildFeedbackPreview(context.Background(), feedbackInput(), key)
	if e != nil {
		t.Fatal(e)
	}
	if a.State != "local_review_export_only" || a.ObservationCount != 2 || a.Proposal == nil || a.Proposal.DistinctAncestryCount != 2 ||
		a.Proposal.Authority != "not_accepted_not_transmitted" || a.Proposal.ProposalID != b.Proposal.ProposalID {
		t.Fatalf("not a stable proposal %#v", a)
	}
	data, _ := json.Marshal(a)
	if bytes.Contains(data, []byte("PRIVATE FIXTURE")) || bytes.Contains(data, []byte("111111111111")) ||
		bytes.Contains(data, key) || bytes.Contains(data, []byte("fictional animation")) {
		t.Fatalf("private raw source or query exposed in preview: %s", data)
	}
	for _, ev := range a.Proposal.Evidence {
		if !shaEvidenceRef(ev.EvidenceRef) || !shaEvidenceRef(ev.AncestryRef) || ev.Source != "bitmagnet" {
			t.Fatalf("bad privacy reference %#v", ev)
		}
	}
	c, e := buildFeedbackPreview(context.Background(), feedbackInput(), []byte(strings.Repeat("Q", 32)))
	if e != nil || c.Proposal.ProposalID == a.Proposal.ProposalID {
		t.Fatal("keyed privacy boundaries were not separated", e)
	}
}

func TestFeedbackNoHitsIsNotConceptAbsence(t *testing.T) {
	feedbackFixture(t, `{"data":{"torrentContent":{"search":{"hasNextPage":false,"items":[]}}}}`)
	x, e := buildFeedbackPreview(context.Background(), feedbackInput(), []byte(strings.Repeat("X", 32)))
	if e != nil || x.Proposal != nil || x.State != "no_matching_observation_not_concept_absence" || x.ObservationCount != 0 {
		t.Fatalf("wrong no-hit semantics: %#v %v", x, e)
	}
}

func TestFeedbackPartialUpstreamNeverPromotesCandidate(t *testing.T) {
	feedbackFixture(t, `{"errors":[{"message":"synthetic schema failure"}]}`)
	x, e := buildFeedbackPreview(context.Background(), feedbackInput(), []byte(strings.Repeat("X", 32)))
	if e != nil || x.Proposal != nil || x.State != "source_partial_or_failed_no_proposal" {
		t.Fatalf("partial upstream accepted as candidate %#v %v", x, e)
	}
}

func TestFeedbackRefusesWebConsentAndTooShortKey(t *testing.T) {
	feedbackFixture(t, `{"data":{"torrentContent":{"search":{"items":[]}}}}`)
	in := feedbackInput()
	in.Query.IncludeWeb = true
	if _, e := buildFeedbackPreview(context.Background(), in, []byte(strings.Repeat("X", 32))); e == nil {
		t.Fatal("external web egress accepted in local feedback")
	}
	in.Query.IncludeWeb = false
	if _, e := buildFeedbackPreview(context.Background(), in, []byte("short")); e == nil {
		t.Fatal("low entropy privacy key accepted")
	}
}

func TestFeedbackHTTPNeedsKeyAndRejectsUnknownFields(t *testing.T) {
	feedbackFixture(t, `{"data":{"torrentContent":{"search":{"items":[]}}}}`)
	mux := http.NewServeMux()
	registerProposals(mux)
	body := []byte(`{"query":{"query":"fictional animation"},"proposal_kind":"coverage_gap","candidate_label":"fictional","language":"en"}`)
	request := func(payload []byte) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/api/v1/discovery/feedback/preview", bytes.NewReader(payload))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}
	t.Setenv("BCE_DISCOVERY_HMAC_KEY", "")
	if got := request(body); got.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing key not refused: %d", got.Code)
	}
	t.Setenv("BCE_DISCOVERY_HMAC_KEY", strings.Repeat("M", 32))
	if got := request(append(body[:len(body)-1], []byte(`,"user_supplied_source_url":"http://example.com"}`)...)); got.Code != http.StatusBadRequest {
		t.Fatalf("unexpected fields admitted: %d, %s", got.Code, got.Body.String())
	}
	if got := request(body); got.Code != http.StatusOK || !strings.Contains(got.Body.String(), "no_matching_observation_not_concept_absence") {
		t.Fatalf("valid no-hit HTTP outcome missing %d %s", got.Code, got.Body.String())
	}
}
