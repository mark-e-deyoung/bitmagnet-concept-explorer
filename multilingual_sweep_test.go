package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const sweepFixture = `{"schema_version":"bce-graph-preview-v1","snapshot_id":"fictional-pinned","concepts":[{"id":"media.animation","label":"Animation","lexicalizations":[{"language":"en","term":"animated film"},{"language":"de","term":"Animation"},{"language":"th","term":"แอนิเมชัน"},{"language":"ja","term":"アニメーション"},{"language":"zh","term":"动画"}]},{"id":"media.movie","label":"Movie","lexicalizations":[{"language":"de","term":"Film"}]}]}`

func TestSweepPlanLanguageSelectionBudgetAndDedup(t *testing.T) {
	setupFixtureGraph(t, sweepFixture)
	p, e := makeSweepPlan(SweepRequest{Query: "Animation", ConceptIDs: []string{"media.animation"}, Languages: []string{"de", "th", "ja"}, MaxBitmagnetQueries: 3})
	if e != nil {
		t.Fatal(e)
	}
	if p.Authority != "digest-pinned-preview-not-cpe-accepted" || len(p.Bitmagnet) != 3 || len(p.Searxng) != 0 || p.ExternalAdmitted {
		t.Fatalf("unsafe plan: %#v", p)
	}
	if p.Bitmagnet[0].Term != "Animation" || p.Bitmagnet[1].Language != "th" || p.Bitmagnet[2].Language != "ja" {
		t.Fatalf("dedup or lexical language was lost: %#v", p.Bitmagnet)
	}
	if len(p.Omitted) != 0 {
		t.Fatalf("unexpected omissions: %v", p.Omitted)
	}
	if p.ProjectionDigest == "" {
		t.Fatal("projection digest missing")
	}
}
func TestExternalSweepRequiresConsentOperatorAndPrivateTarget(t *testing.T) {
	setupFixtureGraph(t, sweepFixture)
	req := SweepRequest{Query: "animation", ConceptIDs: []string{"media.animation"}, Languages: []string{"en", "de"}, IncludeSearxng: true}
	if _, e := makeSweepPlan(req); e == nil {
		t.Fatal("missing consent admitted")
	}
	req.ExternalConsent = true
	if _, e := makeSweepPlan(req); e == nil {
		t.Fatal("missing operator admission accepted")
	}
	t.Setenv("BCE_SEARXNG_SWEEP_ALLOWED", "true")
	t.Setenv("BCE_SEARXNG_URL", "https://search.example.com/search")
	if _, e := makeSweepPlan(req); e == nil {
		t.Fatal("public SearXNG endpoint automatically swept")
	}
	t.Setenv("BCE_SEARXNG_URL", "http://127.0.0.1:8090/search")
	plan, e := makeSweepPlan(req)
	if e != nil {
		t.Fatal(e)
	}
	if !plan.ExternalAdmitted || len(plan.Searxng) != 3 {
		t.Fatalf("unexpected private search plan %#v", plan)
	}
	for _, q := range plan.Searxng {
		if !safeExternalSweepTerm(q.Term) {
			t.Fatalf("unsafe term: %q", q.Term)
		}
	}
}
func TestSweepRejectsBangSyntaxAndUnsafeLanguages(t *testing.T) {
	for _, term := range []string{"!!duckduckgo", "!google topic", ":de forced-language", "film\ninjection"} {
		if safeExternalSweepTerm(term) {
			t.Errorf("unsafe SearXNG query accepted %q", term)
		}
	}
	for _, lang := range []string{"../../secret", "eng;DROP", "x", ""} {
		if sweepLanguage(lang) {
			t.Errorf("invalid language accepted %q", lang)
		}
	}
	if !sweepLanguage("zh-CN") || !sweepLanguage("th") {
		t.Fatal("valid language code rejected")
	}
}
func TestSweepExecuteUsesOneLocalQueryPlanAndScopedExternalSearch(t *testing.T) {
	setupFixtureGraph(t, sweepFixture)
	var localRequests, externalRequests atomic.Int32
	bitmagnet := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		localRequests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"torrentContent":{"search":{"hasNextPage":false,"items":[{"title":"synthetic metadata","contentType":"movie","torrent":{"infoHash":"abc111"}}]}}}}`))
	}))
	defer bitmagnet.Close()
	searx := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		externalRequests.Add(1)
		if r.URL.Query().Get("format") != "json" {
			t.Error("missing JSON")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"title":"Synthetic independent reference","url":"https://example.org/","engines":["synthetic"]}],"suggestions":["synthetic suggested alternate","animation",{"unexpected":"ignore"}]}`))
	}))
	defer searx.Close()
	t.Setenv("BCE_BITMAGNET_GRAPHQL", bitmagnet.URL+"/graphql")
	t.Setenv("BCE_SEARXNG_URL", searx.URL+"/search")
	t.Setenv("BCE_SEARXNG_SWEEP_ALLOWED", "true")
	out, e := runSweep(context.Background(), SweepRequest{Query: "animation", ConceptIDs: []string{"media.animation"}, Languages: []string{"en", "th"},
		IncludeSearxng: true, ExternalConsent: true, MaxBitmagnetQueries: 3, MaxSearxngQueries: 2})
	if e != nil {
		t.Fatal(e)
	}
	if out.State != "complete" || out.CPEWrite || out.Bitmagnet == nil || len(out.External) != 2 {
		t.Fatalf("unexpected sweep result %#v", out)
	}
	if localRequests.Load() != 3 || externalRequests.Load() != 2 {
		t.Fatalf("unexpected query count local=%d external=%d", localRequests.Load(), externalRequests.Load())
	}
	if out.External[0].Term != "animation" {
		t.Fatalf("non-deterministic external direct term %v", out.External)
	}
	if out.External[1].Language != "und" && out.External[1].Language != "en" {
		t.Fatalf("lost language metadata %#v", out.External)
	}
	if len(out.ResearchProposals) != 1 || out.ResearchProposals[0].CandidateLabel != "synthetic suggested alternate" ||
		out.ResearchProposals[0].Language != "en" || out.ResearchProposals[0].State != "local-review-export-only" {
		t.Fatalf("suggestions not quarantined or not deduplicated: %#v", out.ResearchProposals)
	}

}
func TestSearXNGProvider429StopsSweepAndSetsCooldown(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "600")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer s.Close()
	t.Setenv("BCE_SEARXNG_URL", s.URL+"/search")
	_, e := searchSearXNGScoped(context.Background(), "synthetic", "de")
	if !errors.Is(e, errSearXNGBackoff) || !strings.Contains(e.Error(), "429") {
		t.Fatalf("429 not respected: %v", e)
	}
	_, e = searchSearXNGScoped(context.Background(), "synthetic", "de")
	if !errors.Is(e, errSearXNGBackoff) || !strings.Contains(e.Error(), "cooling down") {
		t.Fatalf("ignored provider cooldown: %v", e)
	}
}
func TestSweepPlanDoesNotEmitCPEWritesOrRawProposals(t *testing.T) {
	setupFixtureGraph(t, sweepFixture)
	p, e := makeSweepPlan(SweepRequest{Query: "fictional", ConceptIDs: []string{"media.movie"}})
	if e != nil {
		t.Fatal(e)
	}
	b, e := json.Marshal(p)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(string(b), `"mutation_allowed":true`) {
		t.Fatal("planner invented mutation permission")
	}
}

func TestSearXNGConservativeHourBudgetBlocksFourthRequest(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer s.Close()
	t.Setenv("BCE_SEARXNG_URL", s.URL+"/search")
	for i := 0; i < 3; i++ {
		if _, e := searchSearXNGScoped(context.Background(), "fictional", "en"); e != nil {
			t.Fatalf("within hourly budget %d: %v", i, e)
		}
	}
	_, e := searchSearXNGScoped(context.Background(), "fictional", "en")
	if !errors.Is(e, errSearXNGBackoff) || !strings.Contains(e.Error(), "hourly") {
		t.Fatalf("4th automated API call was not refused: %v", e)
	}
}

func TestSweepEvidenceIsOpaqueAndCanStageCPEProposal(t *testing.T) {
	out := SweepResult{
		Bitmagnet: &Execution{Results: []Observation{{InfoHash: "fixture-1", Title: "Private synthetic title"}}},
		External:  []SweepExternalObservation{{Language: "de", Hits: []ExternalObservation{{Title: "External synthetic source", URL: "https://example.org/doc"}}}},
	}
	addSweepEvidence(&out)
	if len(out.Evidence) != 2 {
		t.Fatalf("wrong source evidence references %#v", out.Evidence)
	}
	for _, v := range out.Evidence {
		if !shaEvidenceRef(v.EvidenceRef) || !shaEvidenceRef(v.AncestryRef) {
			t.Fatal("non-opaque source evidence")
		}
		if strings.Contains(v.EvidenceRef, "Private") || strings.Contains(v.AncestryRef, "example.org") {
			t.Fatal("private metadata escaped")
		}
	}
	candidate, e := previewSweepDiscovery(SweepDiscoveryRequest{Kind: "lexicalization_candidate", CandidateLabel: "Beispiel", Language: "de", Evidence: out.Evidence})
	if e != nil || candidate.State != "local-review-export-only" || candidate.Authority != "not_accepted_not_transmitted" {
		t.Fatalf("invalid staged CPE proposal %#v %v", candidate, e)
	}
}

func TestSearXNGRetryAfterLargerThanDefaultIsHonored(t *testing.T) {
	endpoint := "synthetic-example-no-network://429-with-retry-after"
	searXNGBackoff(endpoint, 429, "3600")
	searXNGPace.Lock()
	until := searXNGPace.BlockedUntil[endpoint]
	searXNGPace.Unlock()
	if d := time.Until(until); d < 59*time.Minute || d > 61*time.Minute {
		t.Fatalf("longer Retry-After not honored: %v", d)
	}
}
