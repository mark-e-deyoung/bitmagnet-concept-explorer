package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearXNGExplicitBoundedQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" || r.URL.Query().Get("q") != "fictional" || r.URL.Query().Get("format") != "json" {
			t.Errorf("unexpected SearXNG query %s", r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"title":"Fictional documentation","url":"https://example.org/test","engines":["fixture"]},{"title":"Skipped local link","url":"file:///etc/passwd"}]}`))
	}))
	defer server.Close()
	t.Setenv("BCE_SEARXNG_URL", server.URL+"/search")
	hits, e := searchSearXNG(context.Background(), "fictional")
	if e != nil || len(hits) != 1 || hits[0].Source != "searxng" || hits[0].Engines[0] != "fixture" {
		t.Fatalf("unexpected result %#v %v", hits, e)
	}
}
func TestSearXNGDeniedOrUnavailable(t *testing.T) {
	for _, ep := range []string{"", "https://user:secret@example.org/search", "https://example.org/?x=1", "http://127.0.0.1:1234/private"} {
		t.Setenv("BCE_SEARXNG_URL", ep)
		if _, e := searXNGEndpoint(); e == nil {
			t.Errorf("accepted %q", ep)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(403) }))
	defer server.Close()
	t.Setenv("BCE_SEARXNG_URL", server.URL+"/search")
	if _, e := searchSearXNG(context.Background(), "fictional"); e == nil || !strings.Contains(e.Error(), "403") {
		t.Fatalf("expected JSON disabled failure, got %v", e)
	}
}
func TestSearXNGExecutionSeparation(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"torrentContent":{"search":{"totalCount":1,"items":[{"title":"fictional","contentType":"movie","torrent":{"infoHash":"abcdef012345"}}]}}}}`))
	}))
	defer upstream.Close()
	external := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]string{{"title": "external result", "url": "https://example.org/"}}})
	}))
	defer external.Close()
	t.Setenv("BCE_BITMAGNET_GRAPHQL", upstream.URL+"/graphql")
	t.Setenv("BCE_SEARXNG_URL", external.URL+"/search")
	result, e := execute(context.Background(), Request{Query: "fictional", IncludeWeb: true})
	if e != nil || result.State != "complete" || len(result.Results) != 1 || len(result.ExternalResults) != 1 {
		returnBad(t, result, e)
	}
	if result.Results[0].Title == result.ExternalResults[0].Title {
		t.Fatal("external and Bitmagnet records incorrectly coalesced")
	}
}
func returnBad(t *testing.T, result Execution, e error) {
	t.Helper()
	t.Fatalf("unexpected separated result %#v / %v", result, e)
}
