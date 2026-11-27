package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestMultilingualSweepFeedbackUsesActualLocalObservationsWithOneUnqualifiedAncestry(t *testing.T) {
	setupFixtureGraph(t, sweepFixture)
	feedbackFixture(t, `{"data":{"torrentContent":{"search":{"hasNextPage":false,"items":[{"title":"PRIVATE FIXTURE Topic Variant","contentType":"movie","torrent":{"infoHash":"1111111111111111111111111111111111111111"}}]}}}}`)
	key := []byte(strings.Repeat("X", 32))
	in := FeedbackPreviewRequest{
		Sweep: &SweepRequest{
			Query: "Private Original Topic", ConceptIDs: []string{"media.animation"},
			Languages: []string{"de", "th"}, MaxBitmagnetQueries: 3,
		},
		ProposalKind: "lexicalization_candidate",
		CandidateLabel: "fiktive neue Bezeichnung", Language: "de",
	}
	first, err := buildFeedbackSweepPreview(context.Background(), in, key)
	if err != nil { t.Fatal(err) }
	second, err := buildFeedbackSweepPreview(context.Background(), in, key)
	if err != nil { t.Fatal(err) }
	if first.State != "local_review_export_only" || first.ObservationCount != 1 ||
		first.Proposal == nil || first.Proposal.DistinctAncestryCount != 1 ||
		first.Proposal.Authority != "not_accepted_not_transmitted" ||
		first.Proposal.ProposalID != second.Proposal.ProposalID ||
		first.Proposal.SchemaVersion != "cpe-discovery-proposal-research-v1" {
		t.Fatalf("unexpected research-only proposal: %#v", first)
	}
	raw, err := json.Marshal(first)
	if err != nil { t.Fatal(err) }
	for _, forbidden := range []string{"PRIVATE FIXTURE", "Private Original Topic", strings.Repeat("1", 40), "bce-sweep-source-suggestion-v1"} {
		if bytes.Contains(raw, []byte(forbidden)) { t.Fatalf("raw source escaped: %s", forbidden) }
	}
	different, err := buildFeedbackSweepPreview(context.Background(), in, []byte(strings.Repeat("Y", 32)))
	if err != nil || different.Proposal == nil || different.Proposal.ProposalID == first.Proposal.ProposalID {
		t.Fatalf("source refs not separated by local privacy key: %v", err)
	}
}

func TestMultilingualSweepFeedbackConsentGatesAndSourceFamilyConservatism(t *testing.T) {
	feedbackFixture(t, `{"data":{"torrentContent":{"search":{"hasNextPage":false,"items":[{"title":"PRIVATE FIXTURE local","contentType":"movie","torrent":{"infoHash":"2222222222222222222222222222222222222222"}}]}}}}`)
	var calls atomic.Int32
	ext := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type","application/json")
		_,_ = w.Write([]byte(`{"results":[{"title":"PRIVATE EXTERNAL TITLE","url":"https://example.org/original-work","engines":["fixture"]}],"suggestions":["synthetic language variant"]}`))
	}))
	defer ext.Close()
	t.Setenv("BCE_SEARXNG_SWEEP_ALLOWED", "true")
	t.Setenv("BCE_SEARXNG_URL", ext.URL+"/search")
	isolateSyntheticSearXNGPace(t, ext.URL+"/search")
	key := []byte(strings.Repeat("K", 32))
	in := FeedbackPreviewRequest{
		Sweep: &SweepRequest{Query:"fictional",Languages:[]string{"de"},IncludeSearxng:true,
			MaxBitmagnetQueries:1,MaxSearxngQueries:1},
		ProposalKind:"coverage_gap",CandidateLabel:"test research-only gap",Language:"de",
	}
	if _, err := buildFeedbackSweepPreview(context.Background(), in, key); err == nil || calls.Load()!=0 {
		t.Fatalf("external source invoked without explicit sweep consent: %v",err)
	}
	in.Sweep.ExternalConsent = true
	out, err := buildFeedbackSweepPreview(context.Background(), in, key)
	if err != nil || calls.Load()!=1 || out.State!="local_review_export_only" ||
		out.Proposal==nil || out.Proposal.DistinctAncestryCount!=1 || out.ObservationCount < 2 {
		t.Fatalf("incorrect consent-gated cross-source preview %#v / %v",out,err)
	}
	bySource:=map[string]int{}
	for _,ev:=range out.Proposal.Evidence {
		bySource[ev.Source]++
		if !shaEvidenceRef(ev.EvidenceRef)||!shaEvidenceRef(ev.AncestryRef) {
			t.Fatal("unkeyed or malformed returned source evidence")
		}
	}
	if bySource["bitmagnet"]==0 || bySource["searxng"]==0 {
		t.Fatalf("lost provenance for source-conservative research preview %#v",bySource)
	}
	raw,_ := json.Marshal(out)
	if bytes.Contains(raw,[]byte("PRIVATE EXTERNAL")) || bytes.Contains(raw,[]byte("example.org")) || bytes.Contains(raw,[]byte("2222222222")) {
		t.Fatal("external URL/title or local infohash exposed in feedback response")
	}
}

func TestMultilingualSweepFeedbackPartialAndProjectionMismatchesNeverPromote(t *testing.T) {
	setupFixtureGraph(t, sweepFixture)
	feedbackFixture(t, `{"data":{"torrentContent":{"search":{"hasNextPage":false,"items":[{"title":"fictional","contentType":"movie","torrent":{"infoHash":"3333333333333333333333333333333333333333"}}]}}}}`)
	ext:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		w.Header().Set("Retry-After","20");w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer ext.Close()
	t.Setenv("BCE_SEARXNG_SWEEP_ALLOWED","true")
	t.Setenv("BCE_SEARXNG_URL",ext.URL+"/search")
	isolateSyntheticSearXNGPace(t,ext.URL+"/search")
	in:=FeedbackPreviewRequest{
		Sweep:&SweepRequest{Query:"fictional",ConceptIDs:[]string{"media.animation"},Languages:[]string{"de"},
			IncludeSearxng:true,ExternalConsent:true,MaxBitmagnetQueries:1,MaxSearxngQueries:1},
		ProposalKind:"coverage_gap",CandidateLabel:"fictional candidate",Language:"de",
	}
	key:=[]byte(strings.Repeat("X",32))
	p,err:=buildFeedbackSweepPreview(context.Background(),in,key)
	if err!=nil || p.Proposal!=nil || p.State!="source_partial_or_failed_no_proposal" {
		t.Fatalf("external refusal laundered local hit into trusted proposal %#v / %v",p,err)
	}
	in.Projection="sha256:"+strings.Repeat("b",64)
	if _,err:=buildFeedbackSweepPreview(context.Background(),in,key);err==nil{
		t.Fatal("wrong pinned graph digest admitted")
	}
}

func TestMultilingualSweepAndLocalQueryMustNotShareOneFeedbackEnvelope(t *testing.T) {
	feedbackFixture(t, `{"data":{"torrentContent":{"search":{"hasNextPage":false,"items":[]}}}}`)
	mux:=http.NewServeMux()
	registerProposals(mux)
	t.Setenv("BCE_DISCOVERY_HMAC_KEY",strings.Repeat("Z",32))
	body:=[]byte(`{"query":{"query":"private"},"sweep":{"query":"private"},"proposal_kind":"coverage_gap","candidate_label":"fictional","language":"en"}`)
	req:=httptest.NewRequest(http.MethodPost,"/api/v1/discovery/feedback/preview",bytes.NewReader(body))
	rec:=httptest.NewRecorder()
	mux.ServeHTTP(rec,req)
	if rec.Code!=422 || !strings.Contains(rec.Body.String(),"exactly one"){
		t.Fatalf("conflicting sweep and local request admitted: %d %s",rec.Code,rec.Body.String())
	}
}
