package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearXNGRejectsTrailingAndOversizedJSON(t *testing.T) {
	oversized := append([]byte(`{"results":[]}`), bytes.Repeat([]byte(" "), searXNGResponseMaxBytes)...)
	cases := []struct {
		name string
		body []byte
	}{
		{name: "trailing-json", body: []byte(`{"results":[]} {"results":[{"title":"extra","url":"https://example.invalid"}]}`)},
		{name: "valid-prefix-over-budget", body: oversized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write(tc.body)
			}))
			defer server.Close()
			t.Setenv("BCE_SEARXNG_URL", server.URL+"/search")
			got, err := searchSearXNGDetailed(context.Background(), "synthetic query", "")
			if err == nil || len(got.Hits) != 0 {
				t.Fatalf("invalid response accepted as observations: %#v err=%v", got, err)
			}
		})
	}
}
