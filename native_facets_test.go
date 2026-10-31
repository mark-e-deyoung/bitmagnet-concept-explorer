package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildNativeFacetQueryPushdownAndBudget(t *testing.T) {
	r := FacetSearchRequest{Query: "fictional", Filters: QueryFilters{Languages: []string{"de"}, ContentTypes: []string{"ebook"}}, Groups: []string{"releaseYear", "language"}}
	gql, input, e := buildNativeFacetQuery(r)
	if e != nil {
		t.Fatal(e)
	}
	if input["limit"] != 0 || input["totalCount"] != true || input["hasNextPage"] != false {
		t.Fatalf("unexpected bounded aggregation plan %#v", input)
	}
	if input["aggregationBudget"] != nativeAggregationBudget {
		t.Fatalf("missing budget %#v", input)
	}
	if strings.Index(gql, "language {") > strings.Index(gql, "releaseYear {") {
		t.Fatalf("unstable field order: %q", gql)
	}
	facets := input["facets"].(map[string]any)
	lang := facets["language"].(map[string]any)
	if lang["aggregate"] != true || len(lang["filter"].([]string)) != 1 {
		t.Fatalf("no merged native language filter+aggregation %#v", facets)
	}
	if _, ok := facets["contentType"]; !ok {
		t.Fatal("lost content type pushdown")
	}
}
func TestNativeFacetPreservesEstimatesAndYearScalars(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/graphql" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var payload struct {
			Variables struct {
				Input map[string]any `json:"input"`
			} `json:"variables"`
		}
		if e := json.NewDecoder(r.Body).Decode(&payload); e != nil {
			t.Error(e)
		}
		if payload.Variables.Input["totalCount"] != true || payload.Variables.Input["limit"] != float64(0) {
			t.Errorf("unexpected GraphQL input %#v", payload.Variables.Input)
		}
		_, _ = w.Write([]byte(`{"data":{"torrentContent":{"search":{"totalCount":143,"totalCountIsEstimate":true,"aggregations":{"language":[{"value":"de","label":"German","count":11,"isEstimate":false}],"releaseYear":[{"value":2024,"label":"2024","count":50,"isEstimate":true}]}}}}}`))
	}))
	defer upstream.Close()
	t.Setenv("BCE_BITMAGNET_GRAPHQL", upstream.URL+"/graphql")
	got, e := fetchNativeFacets(context.Background(), FacetSearchRequest{Query: "fictional", Groups: []string{"releaseYear", "language"}})
	if e != nil {
		t.Fatal(e)
	}
	if !got.TotalCountIsEstimate || got.TotalCount != 143 || got.Scope != "native_corpus_metadata_aggregate_not_cpe_concepts" {
		t.Fatalf("incorrect total or source: %#v", got)
	}
	if string(got.Groups["releaseYear"][0].Value) != "2024" || !got.Groups["releaseYear"][0].IsEstimate {
		t.Fatalf("lost numeric year or estimate: %#v", got.Groups)
	}
	if got.Groups["language"][0].IsEstimate {
		t.Fatal("estimated flag wrongly inferred from overall total")
	}
}
func TestNativeFacetRejectsResidualPolicyOrUnsupportedGroup(t *testing.T) {
	cases := []FacetSearchRequest{
		{Query: "fixture", Groups: []string{"notAField"}},
		{Query: "fixture", Groups: []string{"language", "language"}},
		{Query: "fixture", Groups: []string{"language"}, Filters: QueryFilters{AdultMetadata: "no"}},
		{Query: "fixture", Groups: []string{"language"}, Filters: QueryFilters{MinSeeders: intp(10)}},
	}
	for _, q := range cases {
		if _, _, e := buildNativeFacetQuery(q); e == nil {
			t.Errorf("accepted unsafe/unsupported native facet request %#v", q)
		}
	}
}
func TestNativeFacetGraphQLErrorNotEmptyResult(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errors":[{"message":"invalid facet"}]}`))
	}))
	defer s.Close()
	t.Setenv("BCE_BITMAGNET_GRAPHQL", s.URL+"/graphql")
	if _, e := fetchNativeFacets(context.Background(), FacetSearchRequest{Query: "fictional", Groups: []string{"language"}}); e == nil {
		t.Fatal("GraphQL error became empty valid facet set")
	}
}

func TestNativeFacetRequiresPresentNonnegativeTotalsButAcceptsExplicitZero(t *testing.T) {
	cases := []struct {
		name string
		body string
		want int
		ok   bool
	}{
		{name: "missing-count", body: `{"data":{"torrentContent":{"search":{"totalCountIsEstimate":false,"aggregations":{"language":[]}}}}}`},
		{name: "missing-estimate-flag", body: `{"data":{"torrentContent":{"search":{"totalCount":0,"aggregations":{"language":[]}}}}}`},
		{name: "null-count", body: `{"data":{"torrentContent":{"search":{"totalCount":null,"totalCountIsEstimate":false,"aggregations":{"language":[]}}}}}`},
		{name: "null-estimate-flag", body: `{"data":{"torrentContent":{"search":{"totalCount":0,"totalCountIsEstimate":null,"aggregations":{"language":[]}}}}}`},
		{name: "negative-count", body: `{"data":{"torrentContent":{"search":{"totalCount":-1,"totalCountIsEstimate":false,"aggregations":{"language":[]}}}}}`},
		{name: "explicit-zero", body: `{"data":{"torrentContent":{"search":{"totalCount":0,"totalCountIsEstimate":false,"aggregations":{"language":[]}}}}}`, want: 0, ok: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			t.Setenv("BCE_BITMAGNET_GRAPHQL", server.URL+"/graphql")
			got, err := fetchNativeFacets(context.Background(), FacetSearchRequest{Query: "synthetic", Groups: []string{"language"}})
			if !tc.ok {
				if err == nil {
					t.Fatalf("invalid totals accepted as exact zero: %#v", got)
				}
				return
			}
			if err != nil || got.TotalCount != tc.want || got.TotalCountIsEstimate {
				t.Fatalf("explicit zero result=%#v err=%v", got, err)
			}
		})
	}
}

func TestNativeFacetBucketsRequirePresentNonnegativeCountsAndEstimateFlags(t *testing.T) {
	cases := []struct {
		name    string
		bucket  string
		wantErr bool
	}{
		{name: "missing-count", bucket: `{"value":"en","label":"English","isEstimate":false}`, wantErr: true},
		{name: "null-count", bucket: `{"value":"en","label":"English","count":null,"isEstimate":false}`, wantErr: true},
		{name: "missing-estimate", bucket: `{"value":"en","label":"English","count":0}`, wantErr: true},
		{name: "null-estimate", bucket: `{"value":"en","label":"English","count":0,"isEstimate":null}`, wantErr: true},
		{name: "negative-count", bucket: `{"value":"en","label":"English","count":-1,"isEstimate":false}`, wantErr: true},
		{name: "explicit-zero", bucket: `{"value":"en","label":"English","count":0,"isEstimate":false}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"data":{"torrentContent":{"search":{"totalCount":0,"totalCountIsEstimate":false,"aggregations":{"language":[` + tc.bucket + `]}}}}}`))
			}))
			defer server.Close()
			t.Setenv("BCE_BITMAGNET_GRAPHQL", server.URL+"/graphql")
			got, err := fetchNativeFacets(context.Background(), FacetSearchRequest{Query: "synthetic", Groups: []string{"language"}})
			if tc.wantErr {
				if err == nil {
					t.Fatalf("incomplete bucket accepted as zero: %#v", got)
				}
				return
			}
			if err != nil || got.Groups["language"][0].Count != 0 || got.Groups["language"][0].IsEstimate {
				t.Fatalf("explicit zero bucket result=%#v err=%v", got, err)
			}
		})
	}
}
