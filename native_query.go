package main

import "strings"

// Only fields backed by pinned upstream Bitmagnet v0.10.0 GraphQL schema
// are pushed down. Source adult metadata and numeric minimums are not native
// facets; these remain explicitly page-local in this version.
var nativeLanguages = map[string]bool{
	"af": true, "ar": true, "az": true, "be": true, "bg": true, "bs": true, "ca": true, "ce": true,
	"co": true, "cs": true, "cy": true, "da": true, "de": true, "el": true, "en": true, "es": true,
	"et": true, "eu": true, "fa": true, "fi": true, "fr": true, "he": true, "hi": true, "hr": true,
	"hu": true, "hy": true, "id": true, "is": true, "it": true, "ja": true, "ka": true, "ko": true,
	"ku": true, "lt": true, "lv": true, "mi": true, "mk": true, "ml": true, "mn": true, "ms": true,
	"mt": true, "nl": true, "no": true, "pl": true, "pt": true, "ro": true, "ru": true, "sa": true,
	"sk": true, "sl": true, "sm": true, "so": true, "sr": true, "sv": true, "ta": true, "th": true,
	"tr": true, "uk": true, "vi": true, "yi": true, "zh": true, "zu": true,
}

func nativeSearchInput(term string, q Request) map[string]any {
	input := map[string]any{"queryString": term, "limit": 20, "totalCount": false, "hasNextPage": true}
	facets := map[string]any{}
	if len(q.Filters.ContentTypes) > 0 {
		facets["contentType"] = map[string]any{"filter": q.Filters.ContentTypes}
	}
	if len(q.Filters.Languages) > 0 {
		values := make([]string, 0, len(q.Filters.Languages))
		for _, l := range q.Filters.Languages {
			values = append(values, strings.ToLower(l))
		}
		facets["language"] = map[string]any{"filter": values}
	}
	if len(facets) > 0 {
		input["facets"] = facets
	}
	field := ""
	descending := false
	switch q.Sort {
	case "seeders_desc":
		field, descending = "seeders", true
	case "seeders_asc":
		field = "seeders"
	case "leechers_desc":
		field, descending = "leechers", true
	case "leechers_asc":
		field = "leechers"
	}
	if field != "" {
		input["orderBy"] = []map[string]any{{"field": field, "descending": descending}}
	}
	return input
}
