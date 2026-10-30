package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

const maxNativeFacetGroups = 4
const nativeAggregationBudget = 10000.0

var nativeAggregateGroups = map[string]bool{
	"contentType": true, "language": true, "genre": true, "releaseYear": true,
	"torrentSource": true, "torrentTag": true, "torrentFileType": true,
	"videoResolution": true, "videoSource": true,
}

// FacetSearchRequest asks for explicit Bitmagnet-side metadata aggregations.
// This is not a CPE policy decision, semantic concept count, or page-local group.
type FacetSearchRequest struct {
	Query   string       `json:"query"`
	Filters QueryFilters `json:"filters,omitempty"`
	Groups  []string     `json:"groups"`
}
type NativeFacetBucket struct {
	Value      json.RawMessage `json:"value"`
	Label      string          `json:"label"`
	Count      int             `json:"count"`
	IsEstimate bool            `json:"is_estimate"`
}

// upstreamNativeFacetBucket matches Bitmagnet's GraphQL response casing.
// Keep the public API's snake_case NativeFacetBucket shape separate.
type upstreamNativeFacetBucket struct {
	Value      json.RawMessage `json:"value"`
	Label      string          `json:"label"`
	Count      int             `json:"count"`
	IsEstimate bool            `json:"isEstimate"`
}
type NativeFacetSummary struct {
	Source               string                         `json:"source"`
	Scope                string                         `json:"scope"`
	TotalCount           int                            `json:"total_count"`
	TotalCountIsEstimate bool                           `json:"total_count_is_estimate"`
	Groups               map[string][]NativeFacetBucket `json:"groups"`
	Warnings             []string                       `json:"warnings"`
}

func validateFacetRequest(r FacetSearchRequest) error {
	if len(strings.TrimSpace(r.Query)) == 0 || len([]rune(r.Query)) > 512 {
		return errors.New("query must contain 1-512 characters")
	}
	if len(r.Groups) < 1 || len(r.Groups) > maxNativeFacetGroups {
		return errors.New("choose 1-4 native facet groups")
	}
	seen := map[string]bool{}
	for _, g := range r.Groups {
		if !nativeAggregateGroups[g] || seen[g] {
			return errors.New("unsupported or repeated native facet group")
		}
		seen[g] = true
	}
	if r.Filters.MinSeeders != nil || r.Filters.MinLeechers != nil || r.Filters.AdultMetadata != "" && r.Filters.AdultMetadata != "any" {
		return errors.New("requested residual metadata filter cannot be applied to native corpus aggregations")
	}
	return validateView(Request{Query: r.Query, Filters: r.Filters})
}
func buildNativeFacetQuery(r FacetSearchRequest) (string, map[string]any, error) {
	if e := validateFacetRequest(r); e != nil {
		return "", nil, e
	}
	input := nativeSearchInput(r.Query, Request{Query: r.Query, Filters: r.Filters})
	input["limit"] = 0
	input["totalCount"] = true
	input["hasNextPage"] = false
	input["aggregationBudget"] = nativeAggregationBudget
	groups := append([]string(nil), r.Groups...)
	sort.Strings(groups)
	facets, ok := input["facets"].(map[string]any)
	if !ok {
		facets = map[string]any{}
	}
	parts := make([]string, 0, len(groups))
	for _, g := range groups {
		v, ok := facets[g].(map[string]any)
		if !ok {
			v = map[string]any{}
		}
		v["aggregate"] = true
		facets[g] = v
		parts = append(parts, g+" { value label count isEstimate }")
	}
	input["facets"] = facets
	gql := "query FacetSearch($input: TorrentContentSearchQueryInput!) { torrentContent { search(input: $input) { totalCount totalCountIsEstimate aggregations { " + strings.Join(parts, " ") + " } } } }"
	return gql, input, nil
}
func fetchNativeFacets(ctx context.Context, r FacetSearchRequest) (NativeFacetSummary, error) {
	gql, input, e := buildNativeFacetQuery(r)
	if e != nil {
		return NativeFacetSummary{}, e
	}
	ep, e := endpoint()
	if e != nil {
		return NativeFacetSummary{}, e
	}
	// Aggregations are often more expensive than ordinary term retrieval and
	// must share the very same process-wide source admission budget.
	select {
	case globalBitmagnetSlots <- struct{}{}:
		defer func() { <-globalBitmagnetSlots }()
	case <-ctx.Done():
		return NativeFacetSummary{}, errors.New("native aggregation global admission cancelled")
	}
	payload, e := json.Marshal(map[string]any{"query": gql, "variables": map[string]any{"input": input}})
	if e != nil {
		return NativeFacetSummary{}, e
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, ep, bytes.NewReader(payload))
	if e != nil {
		return NativeFacetSummary{}, e
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }, Transport: &http.Transport{Proxy: nil}}
	resp, e := client.Do(req)
	if e != nil {
		return NativeFacetSummary{}, errors.New("native Bitmagnet aggregation request failed or timed out")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return NativeFacetSummary{}, fmt.Errorf("Bitmagnet aggregation returned HTTP %d", resp.StatusCode)
	}
	var result struct {
		Data struct {
			TorrentContent struct {
				Search struct {
					TotalCount           int                                    `json:"totalCount"`
					TotalCountIsEstimate bool                                   `json:"totalCountIsEstimate"`
					Aggregations         map[string][]upstreamNativeFacetBucket `json:"aggregations"`
				} `json:"search"`
			} `json:"torrentContent"`
		} `json:"data"`
		Errors []json.RawMessage `json:"errors"`
	}
	if e := decodeBoundedJSON(resp.Body, 2<<20, &result); e != nil {
		return NativeFacetSummary{}, errors.New("invalid, trailing or oversized native aggregation response")
	}
	if len(result.Errors) > 0 {
		return NativeFacetSummary{}, errors.New("Bitmagnet aggregation returned GraphQL errors")
	}
	if result.Data.TorrentContent.Search.TotalCount < 0 {
		return NativeFacetSummary{}, errors.New("negative aggregate total count")
	}
	got := result.Data.TorrentContent.Search.Aggregations
	if got == nil {
		return NativeFacetSummary{}, errors.New("missing native aggregation groups")
	}
	ret := map[string][]NativeFacetBucket{}
	for _, name := range r.Groups {
		arr, ok := got[name]
		if !ok {
			return NativeFacetSummary{}, errors.New("native aggregation group missing from response")
		}
		if len(arr) > 256 {
			return NativeFacetSummary{}, errors.New("native aggregation group exceeds output budget")
		}
		for _, v := range arr {
			if v.Count < 0 || len(v.Label) > 256 {
				return NativeFacetSummary{}, errors.New("invalid native aggregation bucket")
			}
		}
		ret[name] = make([]NativeFacetBucket, 0, len(arr))
		for _, v := range arr {
			ret[name] = append(ret[name], NativeFacetBucket{
				Value: v.Value, Label: v.Label, Count: v.Count, IsEstimate: v.IsEstimate,
			})
		}
	}
	return NativeFacetSummary{Source: "bitmagnet", Scope: "native_corpus_metadata_aggregate_not_cpe_concepts", TotalCount: result.Data.TorrentContent.Search.TotalCount, TotalCountIsEstimate: result.Data.TorrentContent.Search.TotalCountIsEstimate, Groups: ret, Warnings: []string{"Bitmagnet native metadata facet counts can be estimates; neither source tags nor counts establish CPE semantic presence or policy dispositions", "Counts are native and scoped to the direct query with its native filters, not globally merged across multilingual expansions"}}, nil
}
func registerNativeFacets(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/query/facets", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 8192)
		defer r.Body.Close()
		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields()
		var req FacetSearchRequest
		if e := d.Decode(&req); e != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid native facet request"})
			return
		}
		if e := d.Decode(new(any)); e != io.EOF {
			writeJSON(w, 400, map[string]string{"error": "trailing native facet request"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()
		out, e := fetchNativeFacets(ctx, req)
		if e != nil {
			writeJSON(w, 422, map[string]string{"error": e.Error()})
			return
		}
		writeJSON(w, 200, out)
	})
}
