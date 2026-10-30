package main

import "testing"

func TestNativeV010FacetsAndOrder(t *testing.T) {
	q := Request{Query: "fictional", Filters: QueryFilters{Languages: []string{"de", "TH"}, ContentTypes: []string{"movie", "audiobook"}}, Sort: "leechers_desc"}
	if e := validateView(q); e != nil {
		t.Fatal(e)
	}
	input := nativeSearchInput("fictional", q)
	facets, ok := input["facets"].(map[string]any)
	if !ok || len(facets) != 2 {
		t.Fatalf("native facets absent: %#v", input)
	}
	lang := facets["language"].(map[string]any)["filter"].([]string)
	if len(lang) != 2 || lang[1] != "th" {
		t.Fatalf("not canonical language list %#v", lang)
	}
	order := input["orderBy"].([]map[string]any)
	if order[0]["field"] != "leechers" || order[0]["descending"] != true {
		t.Fatalf("wrong native sort %#v", order)
	}
}
func TestNativeSortDoesNotInventUnknownGraphQLField(t *testing.T) {
	q := Request{Query: "fictional", Sort: "title_asc", Filters: QueryFilters{MinSeeders: intp(5), AdultMetadata: "yes"}}
	input := nativeSearchInput("fictional", q)
	if _, ok := input["orderBy"]; ok {
		t.Fatal("unexpected title-based native sort with unproven name identity")
	}
	if _, ok := input["facets"]; ok {
		t.Fatal("unsupported numeric/adult facets pushed down")
	}
}
func TestInvalidNativeLanguageRejected(t *testing.T) {
	for _, l := range []string{"eng", "zh-CN", "../../secret", "x"} {
		if e := validateView(Request{Query: "fixture", Filters: QueryFilters{Languages: []string{l}}}); e == nil {
			t.Fatalf("accepted non-native language enum %s", l)
		}
	}
}

func TestNativeQueryUsesCheapTruncationNotUnusedCount(t *testing.T) {
	q := Request{Query: "fixture"}
	input := nativeSearchInput(q.Query, q)
	if input["totalCount"] != false {
		t.Fatalf("unused budgeted count requested: %#v", input)
	}
	if input["hasNextPage"] != true {
		t.Fatalf("native hasNextPage was not requested: %#v", input)
	}
	if input["limit"] != 20 {
		t.Fatalf("query cap changed: %#v", input)
	}
}
