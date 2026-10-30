package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// One process-wide admission ceiling prevents independent HTTP requests or batch
// callers from multiplying the per-request fan-out against the same database.
var globalBitmagnetSlots = make(chan struct{}, 4)

const gqlSearch = `query Search($input: TorrentContentSearchQueryInput!) { torrentContent { search(input: $input) { hasNextPage items { contentType title seeders leechers languages { id name } content { adult } torrent { infoHash name size } } } } }`

type Observation struct {
	InfoHash      string   `json:"info_hash"`
	Title         string   `json:"title"`
	ContentType   string   `json:"content_type,omitempty"`
	Seeders       *int     `json:"seeders"`
	Leechers      *int     `json:"leechers"`
	Languages     []string `json:"languages,omitempty"`
	AdultMetadata *bool    `json:"adult_metadata,omitempty"`
	Tracks        []string `json:"tracks"`
	Terms         []string `json:"matched_queries"`
}
type TermCoverage struct {
	Track       string `json:"track"`
	Term        string `json:"term"`
	Returned    int    `json:"returned"`
	HasNextPage *bool  `json:"has_next_page"`
}
type Execution struct {
	Version         string                `json:"version"`
	ReadOnly        bool                  `json:"read_only"`
	State           string                `json:"state"`
	Results         []Observation         `json:"results"`
	ExternalResults []ExternalObservation `json:"external_results,omitempty"`
	TrackCounts     map[string]int        `json:"track_counts"`
	Failures        []string              `json:"failures"`
	Warnings        []string              `json:"warnings"`
	Coverage        []TermCoverage        `json:"coverage"`
	BeforeFilters   int                   `json:"before_filters"`
	AfterFilters    int                   `json:"after_filters"`
	Groups          []ResultGroup         `json:"groups"`
	FilterScope     string                `json:"filter_scope"`
}
type gqlResult struct {
	Data struct {
		TorrentContent struct {
			Search struct {
				HasNextPage *bool `json:"hasNextPage"`
				Items       []struct {
					Title       string `json:"title"`
					ContentType string `json:"contentType"`
					Seeders     *int   `json:"seeders"`
					Leechers    *int   `json:"leechers"`
					Languages   []struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"languages"`
					Content *struct {
						Adult *bool `json:"adult"`
					} `json:"content"`
					Torrent struct {
						InfoHash string `json:"infoHash"`
						Name     string `json:"name"`
					} `json:"torrent"`
				} `json:"items"`
			} `json:"search"`
		} `json:"torrentContent"`
	} `json:"data"`
	Errors []json.RawMessage `json:"errors"`
}

func endpoint() (string, error) {
	raw := os.Getenv("BCE_BITMAGNET_GRAPHQL")
	if raw == "" {
		return "", errors.New("Bitmagnet endpoint not configured")
	}
	u, e := url.Parse(raw)
	if e != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "/graphql" || u.RawPath != "" {
		return "", errors.New("invalid configured Bitmagnet endpoint (require http(s) /graphql)")
	}
	// The corpus stays on sovereign private infrastructure. Do not allow a
	// misconfigured endpoint to export search terms to arbitrary DNS or cloud.
	host := u.Hostname()
	if host != "localhost" {
		ip := net.ParseIP(host)
		if ip == nil || !(ip.IsLoopback() || ip.IsPrivate()) {
			return "", errors.New("Bitmagnet endpoint must be a literal private or loopback IP")
		}
	}
	return raw, nil
}
func fetchTerm(ctx context.Context, client *http.Client, ep, term string, q Request) ([]Observation, *bool, error) {
	b, e := json.Marshal(map[string]any{"query": gqlSearch, "variables": map[string]any{"input": nativeSearchInput(term, q)}})
	if e != nil {
		return nil, nil, e
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, ep, bytes.NewReader(b))
	if e != nil {
		return nil, nil, e
	}
	req.Header.Set("Content-Type", "application/json")
	res, e := client.Do(req)
	if e != nil {
		return nil, nil, errors.New("Bitmagnet request failed or timed out")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, nil, fmt.Errorf("Bitmagnet returned HTTP %d", res.StatusCode)
	}
	var parsed gqlResult
	// GraphQL can return HTTP 200 with null/missing data or a partial result.
	// A non-null items array is required by the pinned upstream v0.10 schema:
	// never silently turn an invalid response into a "no hits" result.
	if e := decodeBoundedJSON(res.Body, 2<<20, &parsed); e != nil {
		return nil, nil, errors.New("invalid, trailing or oversized Bitmagnet response")
	}
	if len(parsed.Errors) > 0 {
		return nil, nil, errors.New("Bitmagnet GraphQL reported errors")
	}
	if parsed.Data.TorrentContent.Search.Items == nil {
		return nil, nil, errors.New("Bitmagnet GraphQL search missing required items array")
	}
	if len(parsed.Data.TorrentContent.Search.Items) > 20 {
		return nil, nil, errors.New("Bitmagnet GraphQL items exceed per-term retrieval budget")
	}
	out := make([]Observation, 0, len(parsed.Data.TorrentContent.Search.Items))
	for _, item := range parsed.Data.TorrentContent.Search.Items {
		hash := strings.ToLower(strings.TrimSpace(item.Torrent.InfoHash))
		if hash == "" {
			continue
		}
		langs := []string{}
		for _, l := range item.Languages {
			if l.ID != "" {
				langs = append(langs, l.ID)
			}
		}
		var adult *bool
		if item.Content != nil {
			adult = item.Content.Adult
		}
		out = append(out, Observation{InfoHash: hash, Title: item.Title, ContentType: item.ContentType, Seeders: item.Seeders, Leechers: item.Leechers, Languages: langs, AdultMetadata: adult})
	}
	return out, parsed.Data.TorrentContent.Search.HasNextPage, nil
}
func execute(ctx context.Context, q Request) (Execution, error) {
	p, e := makePlan(q)
	if e != nil {
		return Execution{}, e
	}
	ep, e := endpoint()
	if e != nil {
		return Execution{}, e
	}
	type job struct{ track, term string }
	jobs := []job{}
	for _, track := range p.Tracks {
		for _, term := range track.Terms {
			jobs = append(jobs, job{track.Name, term})
		}
	}
	out := Execution{Version: "v0-provisional", ReadOnly: true, State: "complete", Results: []Observation{}, TrackCounts: map[string]int{"direct": 0, "concept": 0}, Failures: []string{}, Warnings: []string{"Unverified caller expansions: concept matches are retrieval evidence, not semantic assessment", "Result counts are capped per query; zero hits do not establish corpus absence"}, Coverage: []TermCoverage{}}
	// Do not forward local Bitmagnet metadata queries through environment HTTP proxies.
	client := &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }, Transport: &http.Transport{Proxy: nil}}
	sem := make(chan struct{}, 4)
	var mu sync.Mutex
	var externalWG sync.WaitGroup
	if q.IncludeWeb {
		externalWG.Add(1)
		go func() {
			defer externalWG.Done()
			external, err := searchSearXNG(ctx, q.Query)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				out.Failures = append(out.Failures, "searxng: "+err.Error())
				return
			}
			out.ExternalResults = external
		}()
	}
	merged := map[string]*Observation{}
	var wg sync.WaitGroup
	for _, j := range jobs {
		wg.Add(1)
		go func(j job) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				mu.Lock()
				out.Failures = append(out.Failures, j.track+": cancelled")
				mu.Unlock()
				return
			}
			select {
			case globalBitmagnetSlots <- struct{}{}:
			case <-ctx.Done():
				mu.Lock()
				out.Failures = append(out.Failures, j.track+": global admission cancelled")
				mu.Unlock()
				return
			}
			rows, more, err := fetchTerm(ctx, client, ep, j.term, q)
			<-globalBitmagnetSlots
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				out.Failures = append(out.Failures, j.track+": "+err.Error())
				return
			}
			out.TrackCounts[j.track] += len(rows)
			out.Coverage = append(out.Coverage, TermCoverage{Track: j.track, Term: j.term, Returned: len(rows), HasNextPage: more})
			for _, v := range rows {
				existing, ok := merged[v.InfoHash]
				if !ok {
					copy := v
					copy.Tracks = []string{}
					copy.Terms = []string{}
					existing = &copy
					merged[v.InfoHash] = existing
				}
				if !contains(existing.Tracks, j.track) {
					existing.Tracks = append(existing.Tracks, j.track)
				}
				if !contains(existing.Terms, j.term) {
					existing.Terms = append(existing.Terms, j.term)
				}
			}
		}(j)
	}
	wg.Wait()
	externalWG.Wait()
	for _, r := range merged {
		sort.Strings(r.Tracks)
		sort.Strings(r.Terms)
		out.Results = append(out.Results, *r)
	}
	sort.Slice(out.Results, func(i, j int) bool { return out.Results[i].InfoHash < out.Results[j].InfoHash })
	sort.Strings(out.Failures)
	sort.Slice(out.Coverage, func(i, j int) bool {
		if out.Coverage[i].Track != out.Coverage[j].Track {
			return out.Coverage[i].Track < out.Coverage[j].Track
		}
		return out.Coverage[i].Term < out.Coverage[j].Term
	})
	out.BeforeFilters = len(out.Results)
	out.Results, out.Groups = viewResults(out.Results, q)
	out.AfterFilters = len(out.Results)
	out.FilterScope = "bounded_returned_results_only"
	out.Warnings = append(out.Warnings, "Content-type and language facets and peer-count sort are pushed to native Bitmagnet v0.10 when requested; numeric thresholds, adult metadata, some sorting and grouping still apply ONLY to fetched capped results and are not full-corpus counts or age/safety decisions")
	if len(out.Failures) > 0 {
		out.State = "partial"
		expected := len(jobs)
		if q.IncludeWeb {
			expected++
		}
		if len(out.Failures) == expected {
			out.State = "failed"
		}
	}
	return out, nil
}
func contains(items []string, v string) bool {
	for _, x := range items {
		if x == v {
			return true
		}
	}
	return false
}
func registerExecution(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/query/execute", func(w http.ResponseWriter, r *http.Request) {
		if _, err := endpoint(); err != nil {
			writeJSON(w, 503, map[string]string{"error": err.Error()})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 16384)
		defer r.Body.Close()
		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields()
		var q Request
		if err := d.Decode(&q); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid request"})
			return
		}
		if err := d.Decode(new(any)); err != io.EOF {
			writeJSON(w, 400, map[string]string{"error": "trailing request data"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		result, err := execute(ctx, q)
		if err != nil {
			writeJSON(w, 422, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, result)
	})
}
