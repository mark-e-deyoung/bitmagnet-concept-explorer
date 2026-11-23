package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

func TestProviderNativeConceptFiltersAreBoundedAndExclusive(t *testing.T) {
	for _, req := range []MediaLookupRequest{
		{Kind: "movie", Providers: []string{"tmdb"}, TMDbGenreIDs: []int{0}},
		{Kind: "movie", Providers: []string{"tmdb"}, TMDbKeywordIDs: []int{-2}},
		{Kind: "movie", Providers: []string{"tmdb"}, TMDbGenreIDs: []int{1, 2, 3, 4}},
		{Kind: "movie", Providers: []string{"jellyfin"}, JellyfinGenres: []string{"example\ninjection"}},
		{Kind: "movie", Providers: []string{"jellyfin"}, JellyfinGenres: []string{"a"}},
		{Kind: "movie", Providers: []string{"tmdb"}, TMDbGenreIDs: []int{12}, Title: "misdirected title"},
		{Kind: "movie", Providers: []string{"tmdb"}, TMDbKeywordIDs: []int{22}, IMDbID: "tt1234567"},
		{Kind: "movie", Providers: []string{"jellyfin"}, JellyfinGenres: []string{"Drama"}, IncludeRelated: true},
	} {
		if err := validateMediaLookup(req); err == nil {
			t.Fatalf("unsafe native concept query accepted %#v", req)
		}
	}
}

func TestTMDbDiscoverOfficialNativeTaxonomyWithoutCPEPromotion(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || r.URL.Path != "/3/discover/movie" ||
			r.URL.Query().Get("with_genres") != "16,878" ||
			r.URL.Query().Get("with_keywords") != "243" ||
			r.URL.Query().Get("page") != "1" ||
			r.URL.Query().Get("language") != "de-DE" ||
			r.Header.Get("Authorization") != "Bearer "+strings.Repeat("T", 32) {
			t.Errorf("unbounded or incorrect TMDb native concept lookup: %s", r.URL)
		}
		_, _ = w.Write([]byte(`{"results":[{"id":51,"title":"Fictional Animated Space Film","release_date":"2025-01-01","genre_ids":[16,878]}],"total_pages":2}`))
	}))
	defer srv.Close()
	t.Setenv("BCE_TMDB_READ_TOKEN", strings.Repeat("T", 32))
	t.Setenv("BCE_TMDB_API_BASE", srv.URL+"/3")
	q := MediaLookupRequest{Kind: "movie", Providers: []string{"tmdb"}, TMDbGenreIDs: []int{16, 878}, TMDbKeywordIDs: []int{243}, Language: "de-DE"}
	a, err := lookupMedia(context.Background(), q)
	if err != nil || a.Outcomes[0].State != "requires_explicit_external_consent" || calls.Load() != 0 {
		t.Fatalf("remote concept request leaked without consent %#v %v", a, err)
	}
	q.AllowExternalLookup = true
	a, err = lookupMedia(context.Background(), q)
	if err != nil || calls.Load() != 1 || a.Outcomes[0].State != "provider_native_concept_results_unmapped" ||
		len(a.Outcomes[0].Candidates) != 1 ||
		a.Outcomes[0].Candidates[0].MatchBasis != "provider_native_concept_candidate_unmapped" ||
		len(a.Outcomes[0].Warnings) == 0 {
		t.Fatalf("native TMDb discover lost provenance/uncertainty %#v %v", a, err)
	}
}

func TestJellyfinGenreQueryRemainsLocalAndSourceNative(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || r.URL.Path != "/Items" ||
			r.URL.Query().Get("Genres") != "Animation,Science Fiction" ||
			r.URL.Query().Get("Limit") != "10" ||
			r.URL.Query().Has("SearchTerm") ||
			r.Header.Get("Authorization") != "MediaBrowser Token="+strconv.Quote(strings.Repeat("J", 32)) {
			t.Errorf("unsafe Jellyfin source genre query %s", r.URL)
		}
		_, _ = w.Write([]byte(`{"Items":[{"Id":"f1f1f1f1f1f1","Name":"Fictional Local Animated Feature","Type":"Movie","ProductionYear":2024,"ProviderIds":{"Tmdb":"777"}}],"TotalRecordCount":1}`))
	}))
	defer srv.Close()
	t.Setenv("BCE_JELLYFIN_BASE_URL", srv.URL)
	t.Setenv("BCE_JELLYFIN_TOKEN", strings.Repeat("J", 32))
	a, err := lookupMedia(context.Background(), MediaLookupRequest{
		Kind: "movie", Providers: []string{"jellyfin"},
		JellyfinGenres: []string{"Animation", "Science Fiction"},
	})
	if err != nil || calls.Load() != 1 || a.Outcomes[0].State != "provider_native_concept_results_unmapped" ||
		len(a.Outcomes[0].Candidates) != 1 ||
		a.Outcomes[0].Candidates[0].MatchBasis != "source_native_genre_candidate_unmapped" {
		t.Fatalf("native Jellyfin genre query not bounded %#v %v", a, err)
	}
}
