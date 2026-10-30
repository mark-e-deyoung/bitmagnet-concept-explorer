package main

import (
	"errors"
	"sort"
	"strings"
)

// QueryFilters describes an exact, metadata-only view over the bounded returned
// Bitmagnet page. It is NOT a policy decision or corpus-wide facet count.
type QueryFilters struct {
	Languages     []string `json:"languages,omitempty"`
	ContentTypes  []string `json:"content_types,omitempty"`
	MinSeeders    *int     `json:"min_seeders,omitempty"`
	MinLeechers   *int     `json:"min_leechers,omitempty"`
	AdultMetadata string   `json:"adult_metadata,omitempty"`
}

type ResultGroup struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

var knownMediaTypes = map[string]bool{
	"movie": true, "tv_show": true, "music": true, "ebook": true,
	"comic": true, "audiobook": true, "game": true, "software": true, "xxx": true,
}

func validateView(q Request) error {
	if len(q.Filters.Languages) > 16 || len(q.Filters.ContentTypes) > 9 {
		return errors.New("too many facet selections")
	}
	for _, l := range q.Filters.Languages {
		if !nativeLanguages[strings.ToLower(l)] {
			return errors.New("unsupported native Bitmagnet v0.10 language facet")
		}
	}
	for _, v := range q.Filters.ContentTypes {
		if !knownMediaTypes[v] {
			return errors.New("unsupported native Bitmagnet media type")
		}
	}
	if q.Filters.MinSeeders != nil && (*q.Filters.MinSeeders < 0 || *q.Filters.MinSeeders > 100000000) {
		return errors.New("min_seeders outside bounds")
	}
	if q.Filters.MinLeechers != nil && (*q.Filters.MinLeechers < 0 || *q.Filters.MinLeechers > 100000000) {
		return errors.New("min_leechers outside bounds")
	}
	switch q.Filters.AdultMetadata {
	case "", "any", "yes", "no", "unknown":
	default:
		return errors.New("invalid adult metadata facet")
	}
	switch q.Sort {
	case "", "info_hash", "seeders_desc", "seeders_asc", "leechers_desc", "leechers_asc", "title_asc", "title_desc":
	default:
		return errors.New("invalid sort")
	}
	switch q.GroupBy {
	case "", "none", "media_type", "language", "adult_metadata", "track":
	default:
		return errors.New("invalid grouping")
	}
	return nil
}
func metadataAdult(v *bool) string {
	if v == nil {
		return "unknown"
	}
	if *v {
		return "yes"
	}
	return "no"
}
func selected(v string, choices []string) bool {
	if len(choices) == 0 {
		return true
	}
	for _, c := range choices {
		if strings.EqualFold(v, c) {
			return true
		}
	}
	return false
}
func includesOne(values, choices []string) bool {
	if len(choices) == 0 {
		return true
	}
	for _, v := range values {
		if selected(v, choices) {
			return true
		}
	}
	return false
}
func viewResults(rows []Observation, q Request) ([]Observation, []ResultGroup) {
	out := make([]Observation, 0, len(rows))
	for _, r := range rows {
		if !selected(r.ContentType, q.Filters.ContentTypes) || !includesOne(r.Languages, q.Filters.Languages) {
			continue
		}
		if q.Filters.MinSeeders != nil && (r.Seeders == nil || *r.Seeders < *q.Filters.MinSeeders) {
			continue
		}
		if q.Filters.MinLeechers != nil && (r.Leechers == nil || *r.Leechers < *q.Filters.MinLeechers) {
			continue
		}
		if v := q.Filters.AdultMetadata; v != "" && v != "any" && metadataAdult(r.AdultMetadata) != v {
			continue
		}
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		switch q.Sort {
		case "seeders_desc", "seeders_asc":
			cmp := compareNullable(a.Seeders, b.Seeders, q.Sort == "seeders_desc")
			if cmp != 0 {
				return cmp < 0
			}
		case "leechers_desc", "leechers_asc":
			cmp := compareNullable(a.Leechers, b.Leechers, q.Sort == "leechers_desc")
			if cmp != 0 {
				return cmp < 0
			}
		case "title_asc", "title_desc":
			l, r := strings.ToLower(a.Title), strings.ToLower(b.Title)
			if l != r {
				if q.Sort == "title_asc" {
					return l < r
				}
				return l > r
			}
		}
		return a.InfoHash < b.InfoHash
	})
	counts := map[string]int{}
	for _, r := range out {
		keys := []string{}
		switch q.GroupBy {
		case "media_type":
			k := r.ContentType
			if k == "" {
				k = "unknown"
			}
			keys = []string{k}
		case "language":
			for _, language := range r.Languages {
				if selected(language, q.Filters.Languages) {
					keys = append(keys, language)
				}
			}
			if len(keys) == 0 {
				if len(r.Languages) == 0 {
					keys = []string{"unknown"}
				}
			}
		case "adult_metadata":
			keys = []string{metadataAdult(r.AdultMetadata)}
		case "track":
			keys = append(keys, r.Tracks...)
		}
		seen := map[string]bool{}
		for _, k := range keys {
			if !seen[k] {
				counts[k]++
				seen[k] = true
			}
		}
	}
	groups := make([]ResultGroup, 0, len(counts))
	for k, c := range counts {
		groups = append(groups, ResultGroup{Key: k, Count: c})
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Count != groups[j].Count {
			return groups[i].Count > groups[j].Count
		}
		return groups[i].Key < groups[j].Key
	})
	return out, groups
}

// compareNullable sorts unknown values last, regardless of requested direction.
func compareNullable(a, b *int, descending bool) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return 1
	}
	if b == nil {
		return -1
	}
	if *a == *b {
		return 0
	}
	if descending {
		if *a > *b {
			return -1
		}
		return 1
	}
	if *a < *b {
		return -1
	}
	return 1
}
