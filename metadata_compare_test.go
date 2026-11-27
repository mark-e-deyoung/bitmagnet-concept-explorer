package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompareMetadataConservativeClaims(t *testing.T) {
	base := ComparisonRecord{Title: "Fictional Example", Kind: "movie", Year: 2025, IDs: map[string]string{"imdb": "tt1234567"}}
	in := MetadataComparisonRequest{
		Reference: base,
		Observations: []ComparisonObservation{
			{Source: "searxng", State: "observed", Record: ComparisonRecord{Title: " fictional   example "}},
			{Source: "tmdb", State: "observed", Record: base},
			{Source: "reference_catalog", State: "observed", Record: ComparisonRecord{Title: base.Title, Year: 1999, IDs: base.IDs}},
			{Source: "searxng", State: "observed", Record: ComparisonRecord{Title: base.Title, IDs: map[string]string{"imdb": "tt7654321"}}},
			{Source: "jellyfin", State: "partial", Record: base},
			{Source: "publisher", State: "no_match"},
			{Source: "searxng", State: "observed", Record: ComparisonRecord{Title: "Unrelated"}},
		},
	}
	out, err := compareMetadata(in)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"title_candidate_unconfirmed", "shared_catalog_id_claim_unverified", "conflicting_claims_review_required", "conflicting_claims_review_required", "source_incomplete_non_diagnostic", "source_incomplete_non_diagnostic", "unknown"}
	if !out.ReadOnly || out.ExternalRequests != 0 || out.IndependentWitnesses != 0 || len(out.Outcomes) != len(want) {
		t.Fatalf("invalid authority or comparison count: %#v", out)
	}
	for i, row := range out.Outcomes {
		if row.Index != i || row.Identity != want[i] || row.Ancestry != "shared_unqualified" {
			t.Fatalf("row %d: %#v", i, row)
		}
	}
	if out.Outcomes[0].Fields["year"] != "unknown" || out.Outcomes[2].Fields["year"] != "difference" {
		t.Fatal("missing or conflicting years misclassified")
	}
	encoded, err := json.Marshal(out)
	if err != nil || strings.Contains(string(encoded), base.Title) || strings.Contains(string(encoded), "tt1234567") {
		t.Fatal("input identity echoed")
	}
}

func TestMetadataComparisonHTTPRejectsUnsafeRequests(t *testing.T) {
	cases := []struct {
		body string
		code int
	}{
		{`{"reference":{"title":"Example"},"observations":[{"source":"searxng","state":"observed","record":{"title":"Example"}}]}`, 200},
		{`{"reference":{"title":"Example"},"observations":[{"source":"searxng","state":"refused","record":{}}]}`, 200},
		{`{"reference":{"title":"Example"},"observations":[{"source":"arbitrary_url","state":"observed","record":{}}]}`, 422},
		{`{"reference":{"title":"Example"},"observations":[{"source":"publisher","state":"observed","record":{"catalog_ids":{"url":"https://example.invalid"}}}]}`, 422},
		{`{"reference":{"title":"Example"},"observations":[{"source":"publisher","state":"observed","record":{"url":"https://example.invalid"}}]}`, 400},
		{`{"reference":{"title":"Example"},"observations":[]}`, 422},
		{`{"reference":{"title":"Example"},"observations":[{"source":"searxng","state":"observed","record":{"title":"x"}}]} {}`, 400},
	}
	app := handler()
	for _, tc := range cases {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/metadata/compare", strings.NewReader(tc.body)))
		if w.Code != tc.code {
			t.Fatalf("status %d want %d: %s", w.Code, tc.code, w.Body.String())
		}
	}
}
