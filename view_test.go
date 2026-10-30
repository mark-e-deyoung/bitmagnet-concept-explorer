package main

import (
	"encoding/json"
	"testing"
)

func intp(v int) *int    { return &v }
func boolp(v bool) *bool { return &v }
func TestMetadataFacetsSortAndGroup(t *testing.T) {
	rows := []Observation{
		{InfoHash: "b", Title: "Alpha", ContentType: "movie", Languages: []string{"en", "de"}, Seeders: intp(11), Leechers: intp(2), AdultMetadata: boolp(false), Tracks: []string{"direct"}},
		{InfoHash: "c", Title: "Beta", ContentType: "audiobook", Languages: []string{"de"}, Seeders: intp(40), Leechers: intp(9), Tracks: []string{"concept"}},
		{InfoHash: "a", Title: "Gamma", ContentType: "movie", Languages: []string{"ja"}, Seeders: nil, Leechers: intp(15), AdultMetadata: boolp(true), Tracks: []string{"direct", "concept"}},
	}
	q := Request{Query: "fixture", Filters: QueryFilters{Languages: []string{"de"}, MinSeeders: intp(10)}, Sort: "seeders_desc", GroupBy: "language"}
	if err := validateView(q); err != nil {
		t.Fatal(err)
	}
	out, groups := viewResults(rows, q)
	if len(out) != 2 || out[0].InfoHash != "c" || out[1].InfoHash != "b" {
		t.Fatalf("unexpected results %#v", out)
	}
	if len(groups) != 1 || groups[0].Key != "de" || groups[0].Count != 2 {
		t.Fatalf("unexpected groups %#v", groups)
	}
	q.Filters.MinLeechers = intp(5)
	out, _ = viewResults(rows, q)
	if len(out) != 1 || out[0].InfoHash != "c" {
		t.Fatalf("leechers filter: %#v", out)
	}
}
func TestUnknownCountsNeverBecomeZero(t *testing.T) {
	rows := []Observation{{InfoHash: "a", Seeders: nil}, {InfoHash: "b", Seeders: intp(0)}, {InfoHash: "c", Seeders: intp(5)}}
	q := Request{Query: "fixture", Sort: "seeders_desc"}
	out, _ := viewResults(rows, q)
	if out[0].InfoHash != "c" || out[1].InfoHash != "b" || out[2].InfoHash != "a" {
		t.Fatalf("unknown order %#v", out)
	}
	q.Filters.MinSeeders = intp(0)
	out, _ = viewResults(rows, q)
	if len(out) != 2 {
		t.Fatalf("unknown incorrectly passed numeric filter %#v", out)
	}
}
func TestAdultMetadataIsNotPolicy(t *testing.T) {
	rows := []Observation{{InfoHash: "a", AdultMetadata: nil}, {InfoHash: "b", AdultMetadata: boolp(false)}, {InfoHash: "c", AdultMetadata: boolp(true)}}
	for flag, want := range map[string]string{"yes": "c", "no": "b", "unknown": "a"} {
		out, _ := viewResults(rows, Request{Query: "fixture", Filters: QueryFilters{AdultMetadata: flag}})
		if len(out) != 1 || out[0].InfoHash != want {
			t.Errorf("flag=%q output=%#v", flag, out)
		}
	}
}
func TestFiltersRejectInvalidAndJSON(t *testing.T) {
	tests := []Request{
		{Query: "x", Filters: QueryFilters{MinSeeders: intp(-1)}},
		{Query: "x", Filters: QueryFilters{ContentTypes: []string{"rom"}}},
		{Query: "x", Sort: "unsafe_direct_sql"},
		{Query: "x", GroupBy: "genre"},
		{Query: "x", Filters: QueryFilters{AdultMetadata: "safe"}},
	}
	for _, q := range tests {
		if err := validateView(q); err == nil {
			t.Errorf("expected validation failure: %#v", q)
		}
	}
	var q Request
	err := json.Unmarshal([]byte(`{"query":"fictional","filters":{"languages":["de"],"content_types":["ebook"],"min_leechers":3},"sort":"leechers_desc","group_by":"media_type"}`), &q)
	if err != nil || q.Filters.MinLeechers == nil || *q.Filters.MinLeechers != 3 || validateView(q) != nil {
		t.Fatalf("bad typed input %#v %v", q, err)
	}
}
