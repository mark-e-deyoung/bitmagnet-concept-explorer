package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestExecuteDualTrackOverlap(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/graphql" {
			t.Errorf("invalid upstream path")
			w.WriteHeader(404)
			return
		}
		var payload struct {
			Variables struct {
				Input struct {
					QueryString string `json:"queryString"`
				} `json:"input"`
			} `json:"variables"`
		}
		if e := json.NewDecoder(r.Body).Decode(&payload); e != nil {
			t.Error(e)
			return
		}
		title := "shared"
		if payload.Variables.Input.QueryString == "other" {
			title = "other"
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"torrentContent":{"search":{"hasNextPage":true,"items":[{"title":"` + title + `","contentType":"OTHER","seeders":2,"torrent":{"infoHash":"abcdef012345","name":"fixture"}}]}}}}`))
	}))
	defer upstream.Close()
	t.Setenv("BCE_BITMAGNET_GRAPHQL", upstream.URL+"/graphql")
	result, e := execute(context.Background(), Request{Query: "sample", Expansions: []Expansion{{Term: "other", Provenance: "synthetic-test"}}})
	if e != nil {
		t.Fatal(e)
	}
	if result.State != "complete" || len(result.Results) != 1 || len(result.Results[0].Tracks) != 2 || len(result.Coverage) != 2 || result.Coverage[0].HasNextPage == nil || !*result.Coverage[0].HasNextPage || !result.ReadOnly {
		t.Fatalf("unexpected result %#v", result)
	}
}
func TestNoEndpoint(t *testing.T) {
	t.Setenv("BCE_BITMAGNET_GRAPHQL", "")
	_, e := execute(context.Background(), Request{Query: "sample"})
	if e == nil || !strings.Contains(e.Error(), "not configured") {
		t.Fatalf("expected configuration failure: %v", e)
	}
}
func TestGraphQLErrorDoesNotCountAsAbsence(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":[{"message":"schema mismatch"}]}`))
	}))
	defer upstream.Close()
	t.Setenv("BCE_BITMAGNET_GRAPHQL", upstream.URL+"/graphql")
	result, e := execute(context.Background(), Request{Query: "example"})
	if e != nil || result.State != "failed" || len(result.Failures) != 1 {
		t.Fatalf("unexpected result %#v / %v", result, e)
	}
}

func TestMissingNativePaginationIsUnknownNotNoMorePages(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"torrentContent":{"search":{"items":[{"title":"synthetic","contentType":"movie","torrent":{"infoHash":"abc123","name":"synthetic"}}]}}}}`))
	}))
	defer upstream.Close()
	t.Setenv("BCE_BITMAGNET_GRAPHQL", upstream.URL+"/graphql")
	out, e := execute(context.Background(), Request{Query: "synthetic"})
	if e != nil || out.State != "complete" || len(out.Coverage) != 1 {
		t.Fatalf("unexpected execution %v %#v", e, out)
	}
	if out.Coverage[0].HasNextPage != nil {
		t.Fatal("null/absent pagination was turned into false")
	}
}

func TestGlobalBitmagnetAdmissionAcrossParallelExecutions(t *testing.T) {
	var running, peak atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := running.Add(1)
		defer running.Add(-1)
		for {
			old := peak.Load()
			if current <= old || peak.CompareAndSwap(old, current) {
				break
			}
		}
		time.Sleep(15 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"torrentContent":{"search":{"hasNextPage":false,"items":[]}}}}`))
	}))
	defer upstream.Close()
	t.Setenv("BCE_BITMAGNET_GRAPHQL", upstream.URL+"/graphql")
	var wg sync.WaitGroup
	errors := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			q := Request{Query: "synthetic", Expansions: []Expansion{{Term: "alpha", Provenance: "synthetic"}, {Term: "beta", Provenance: "synthetic"}}}
			out, e := execute(context.Background(), q)
			if e != nil {
				errors <- e
				return
			}
			if out.State != "complete" {
				errors <- fmt.Errorf("unexpected execution state %s", out.State)
			}
		}()
	}
	wg.Wait()
	close(errors)
	for e := range errors {
		t.Error(e)
	}
	if got := peak.Load(); got > 4 || got < 2 {
		t.Fatalf("global Bitmagnet admission incorrect: peak %d", got)
	}
}

func TestMalformedGraphQLSuccessIsNotZeroHit(t *testing.T) {
	cases := []string{
		`{}`,
		`{"data":null}`,
		`{"data":{"torrentContent":null}}`,
		`{"data":{"torrentContent":{"search":null}}}`,
		`{"data":{"torrentContent":{"search":{}}}}`,
		`{"data":{"torrentContent":{"search":{"items":null}}}}`,
		`{"data":{"torrentContent":{"search":{"items":[]}}}}{}`,
	}
	for _, body := range cases {
		t.Run(fmt.Sprintf("case-%d", len(body)), func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(body))
			}))
			defer upstream.Close()
			t.Setenv("BCE_BITMAGNET_GRAPHQL", upstream.URL+"/graphql")
			out, e := execute(context.Background(), Request{Query: "synthetic"})
			if e != nil || out.State != "failed" || len(out.Failures) != 1 || len(out.Coverage) != 0 || len(out.Results) != 0 {
				t.Fatalf("malformed upstream response became apparent valid search: error %v, result %#v", e, out)
			}
		})
	}
}
