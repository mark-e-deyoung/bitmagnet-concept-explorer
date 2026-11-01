package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearXNGRequiresResultsArrayOnHTTP200(t *testing.T) {
	for _, tc := range []struct {
		name	string
		body	string
		invalid	bool
	}{
		{name: "missing-results", body: `{"suggestions":["unsafe-to-accept-without-results"]}`, invalid: true},
		{name: "null-results", body: `{"results":null}`, invalid: true},
		{name: "explicit-empty-results", body: `{"results":[]}`, invalid: false},
		{name: "valid-single-result", body: `{"results":[{"title":"Synthetic reference","url":"https://example.invalid/reference"}]}`, invalid: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.body))
			}))
			defer upstream.Close()
			t.Setenv("BCE_SEARXNG_URL", upstream.URL+"/search")
			result, err := searchSearXNGDetailed(context.Background(), "synthetic", "")
			if tc.invalid {
				if err == nil || len(result.Hits) != 0 || len(result.Suggestions) != 0 {
					t.Fatalf("malformed provider response treated as usable result: got %#v, err %v", result, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("valid provider results array rejected: %v", err)
			}
			want := 0
			if tc.name == "valid-single-result" {
				want = 1
			}
			if len(result.Hits) != want {
				t.Fatalf("provider hits=%d, want %d", len(result.Hits), want)
			}
		})
	}
}
