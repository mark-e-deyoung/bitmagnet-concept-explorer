package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestBitmagnetEndpointRejectsCloudAndDNS(t *testing.T) {
	for _, tc := range []struct {
		raw string
		ok  bool
	}{
		{"http://127.0.0.1:8080/graphql", true},
		{"http://10.0.0.2:8080/graphql", true},
		{"http://192.168.1.2:8080/graphql", true},
		{"http://[::1]:8080/graphql", true},
		{"http://localhost:8080/graphql", true},
		{"https://8.8.8.8/graphql", false},
		{"http://example.com/graphql", false},
		{"http://169.254.1.4/graphql", false},
		{"https://127.0.0.1/graphql?debug=1", false},
		{"http://user:password@127.0.0.1/graphql", false},
		{"http://127.0.0.1/api/v1/graphql", false},
	} {
		t.Run(strings.ReplaceAll(tc.raw, "/", "_"), func(t *testing.T) {
			t.Setenv("BCE_BITMAGNET_GRAPHQL", tc.raw)
			_, err := endpoint()
			if (err == nil) != tc.ok {
				t.Fatalf("endpoint admission=%v want %v", err == nil, tc.ok)
			}
		})
	}
}
func TestBitmagnetRejectsOversizedNativePage(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		items := make([]map[string]any, 21)
		for i := range items {
			items[i] = map[string]any{"title": "synthetic", "contentType": "movie", "torrent": map[string]string{"infoHash": "abcdef", "name": "fixture"}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"torrentContent": map[string]any{"search": map[string]any{"hasNextPage": true, "items": items}}}})
	}))
	defer upstream.Close()
	t.Setenv("BCE_BITMAGNET_GRAPHQL", upstream.URL+"/graphql")
	out, err := execute(context.Background(), Request{Query: "synthetic"})
	if err != nil || out.State != "failed" || len(out.Coverage) != 0 || len(out.Results) != 0 {
		t.Fatalf("oversized source page accepted: err %v result %#v", err, out)
	}
}
func TestNativeAggregationsShareGlobalBudget(t *testing.T) {
	var calls atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer upstream.Close()
	t.Setenv("BCE_BITMAGNET_GRAPHQL", upstream.URL+"/graphql")
	// Occupy all source slots as if ordinary term searches are in progress.
	// This must prevent expensive native aggregates from bypassing admission.
	for i := 0; i < cap(globalBitmagnetSlots); i++ {
		globalBitmagnetSlots <- struct{}{}
	}
	defer func() {
		for i := 0; i < cap(globalBitmagnetSlots); i++ {
			<-globalBitmagnetSlots
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Millisecond)
	defer cancel()
	_, err := fetchNativeFacets(ctx, FacetSearchRequest{Query: "synthetic", Groups: []string{"language"}})
	if err == nil || !strings.Contains(err.Error(), "admission") {
		t.Fatalf("aggregation was not rejected at exhausted global budget: %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("aggregation reached source without admission: %d requests", calls.Load())
	}
}
