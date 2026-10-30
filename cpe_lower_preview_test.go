package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHistoricalCPELowerPreviewNeverAuthorizesExecution(t *testing.T) {
	var p HistoricalAdapterPlan
	if e := json.Unmarshal([]byte(historicalPlanFixture), &p); e != nil {
		t.Fatal(e)
	}
	got, e := lowerHistoricalCPEPreview(p)
	if e != nil {
		t.Fatal(e)
	}
	if got.ExecutionAuthorized || got.Status != "lowered_preview_not_authorized" || got.SemanticAuthority != "not_accepted_cpe_projection" || len(got.Queries) != 2 {
		t.Fatalf("unsafe preview result: %#v", got)
	}
	for i, q := range got.Queries {
		if q.ConceptID != "media.production.animation" || q.Status != "candidate_retrieval_only" {
			t.Fatalf("missing concept attribution/role: %#v", q)
		}
		if q.NativeInput["totalCount"] != false || q.NativeInput["hasNextPage"] != true || q.NativeInput["limit"] != 20 {
			t.Fatalf("unbounded native retrieval: %#v", q.NativeInput)
		}
		if _, ok := q.NativeInput["facets"]; ok {
			t.Fatal("CPE lexical language was mistaken for source metadata language")
		}
		if !strings.HasPrefix(q.NativeInput["queryString"].(string), `"`) {
			t.Fatal("literal was not quoted")
		}
		if i == 0 && q.Language != "en" || i == 1 && q.Language != "de" {
			t.Fatal("lost source lexical language provenance")
		}
	}
}
func TestHistoricalCPELoweringRejectsFTSOperators(t *testing.T) {
	for _, term := range []string{"animation | !film", "ani*", "movie (2025)", "sci-fi", "alice\\bob", "two  spaces", "word\nnewline", " leading", "trailing ", "\ntrimmed"} {
		if _, e := bitmagnetSafeLiteral(term); e == nil {
			t.Fatalf("unsafe native query accepted: %q", term)
		}
	}
	for _, term := range []string{"Animation", "ภาพยนตร์", "アニメーション", "动画", "Überprüfung", "animated film"} {
		if out, e := bitmagnetSafeLiteral(term); e != nil || len(out) < 2 {
			t.Fatalf("safe literal rejected %q %v", term, e)
		}
	}
}
func TestHistoricalCPELoweringRefusesUnsupportedSemanticShapes(t *testing.T) {
	for _, replacement := range []string{`"hard_negative"`, `"should"`} {
		s := strings.Replace(historicalPlanFixture, `"relation":"must"`, `"relation":`+replacement, 1)
		var p HistoricalAdapterPlan
		if e := json.Unmarshal([]byte(s), &p); e != nil {
			t.Fatal(e)
		}
		if x, e := lowerHistoricalCPEPreview(p); e == nil || x.ExecutionAuthorized {
			t.Fatalf("unsafe relation lowered: %s", replacement)
		}
	}
}
func TestHistoricalCPELowerPreviewHTTPNoNetworkExecution(t *testing.T) {
	t.Setenv("BCE_BITMAGNET_GRAPHQL", "")
	app := httptest.NewServer(handler())
	defer app.Close()
	response, e := http.Post(app.URL+"/api/v1/cpe/adapter-plan/lower-preview", "application/json", bytes.NewBufferString(historicalPlanFixture))
	if e != nil {
		t.Fatal(e)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("unexpected preview response HTTP %d", response.StatusCode)
	}
	var parsed HistoricalLowerPreview
	if e := json.NewDecoder(response.Body).Decode(&parsed); e != nil {
		t.Fatal(e)
	}
	if parsed.ExecutionAuthorized || len(parsed.Queries) != 2 {
		t.Fatalf("unexpected preview %#v", parsed)
	}
}
