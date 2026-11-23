package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

func TestMediaLookupRejectsUnknownProviderAndUnsafeIdentity(t *testing.T) {
	cases := []MediaLookupRequest{
		{Kind: "movie", Title: "fictional", Providers: []string{"arbitrary"}},
		{Kind: "movie", Title: "fictional", Providers: []string{"jellyfin", "jellyfin"}},
		{Kind: "movie", Title: "fictional", Providers: []string{"tmdb"}, Language: "en\nUS"},
		{Kind: "movie", IMDbID: "tt12/../x", Providers: []string{"tmdb"}},
		{Kind: "movie", JellyfinItemID: "../../etc", Providers: []string{"jellyfin"}},
		{Kind: "unknown", Title: "fictional", Providers: []string{"tmdb"}},
		{Kind: "movie", Providers: []string{"tmdb"}},
	}
	for _, q := range cases {
		if err := validateMediaLookup(q); err == nil {
			t.Fatalf("unsafe lookup accepted: %#v", q)
		}
	}
	for _, endpoint := range []string{"http://169.254.169.254", "http://8.8.8.8", "http://example.com", "http://127.0.0.1/prefix"} {
		if _, e := providerURL(endpoint, false); e == nil {
			t.Fatalf("Jellyfin local target constraint bypassed: %s", endpoint)
		}
	}
	for _, endpoint := range []string{"http://evil.example/3", "https://evil.example/3", "https://api.themoviedb.org:8888/3", "https://api.themoviedb.org/2", "http://169.254.169.254/3"} {
		if _, e := providerURL(endpoint, true); e == nil {
			t.Fatalf("TMDb allowlist bypassed: %s", endpoint)
		}
	}
}

func TestTMDbSearchExplicitEgressAndNoSilentIdentity(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/3/search/movie" || r.URL.Query().Get("query") != "Fictional Work" ||
			r.URL.Query().Get("year") != "2025" ||
			r.URL.Query().Get("language") != "de-DE" ||
			r.Header.Get("Authorization") != "Bearer "+strings.Repeat("S", 32) {
			t.Errorf("incorrect bounded query, path, language or token")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":11,"title":"Fictional Work","release_date":"2025-02-03","genre_ids":[12]},{"id":22,"title":"Fictional Work","release_date":"1987-01-01"}],"total_pages":2}`))
	}))
	defer srv.Close()
	t.Setenv("BCE_TMDB_API_BASE", srv.URL+"/3")
	t.Setenv("BCE_TMDB_READ_TOKEN", strings.Repeat("S", 32))
	in := MediaLookupRequest{Kind: "movie", Title: "Fictional Work", Year: 2025, Language: "de-DE", Providers: []string{"tmdb"}}
	first, e := lookupMedia(context.Background(), in)
	if e != nil || first.Outcomes[0].State != "requires_explicit_external_consent" || requests.Load() != 0 {
		t.Fatalf("external title leaked without consent: %#v %v", first, e)
	}
	in.AllowExternalLookup = true
	resp, e := lookupMedia(context.Background(), in)
	if e != nil || requests.Load() != 1 || len(resp.Outcomes) != 1 || len(resp.Outcomes[0].Candidates) != 2 ||
		resp.Outcomes[0].Candidates[0].MatchBasis != "title_candidate_unconfirmed" ||
		len(resp.Outcomes[0].Warnings) == 0 || !resp.ReadOnly {
		t.Fatalf("incorrect ambiguous first-page candidate behavior %#v %v", resp, e)
	}
}

func TestTMDbIMDbIDCrosswalkIsNotDirectIMDbScraping(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Header.Get("Authorization") != "Bearer "+strings.Repeat("T", 32) {
			t.Error("token absent")
		}
		switch r.URL.Path {
		case "/3/find/tt1234567":
			if r.URL.Query().Get("external_source") != "imdb_id" {
				t.Error("incorrect IMDb query")
			}
			_, _ = w.Write([]byte(`{"movie_results":[{"id":44,"title":"Original Fictional Movie","release_date":"1999-12-01"}],"tv_results":[]}`))
		case "/3/movie/44":
			_, _ = w.Write([]byte(`{"id":44,"title":"Original Fictional Movie","release_date":"1999-12-01","genre_ids":[5,19]}`))
		case "/3/movie/44/recommendations":
			_, _ = w.Write([]byte(`{"results":[{"id":45,"title":"Another Fictional Film","release_date":"2000-01-01"}],"total_pages":1}`))
		default:
			t.Errorf("unexpected remote endpoint %s", r.URL)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("BCE_TMDB_API_BASE", srv.URL+"/3")
	t.Setenv("BCE_TMDB_READ_TOKEN", strings.Repeat("T", 32))
	cross := MediaLookupRequest{Kind: "movie", IMDbID: "tt1234567", Providers: []string{"imdb", "tmdb"}, AllowExternalLookup: true}
	a, e := lookupMedia(context.Background(), cross)
	if e != nil || a.Outcomes[0].State != "licensed_source_not_configured" ||
		a.Outcomes[1].Candidates[0].MatchBasis != "provider_imdb_id_crosswalk" || requests.Load() != 1 {
		t.Fatalf("IMDb crosswalk missing or illegal direct source call %#v %v", a, e)
	}
	byID := MediaLookupRequest{Kind: "movie", TMDbID: "44", Providers: []string{"tmdb"}, AllowExternalLookup: true, IncludeRelated: true}
	b, e := lookupMedia(context.Background(), byID)
	if e != nil || len(b.Outcomes[0].Related) != 1 || b.Outcomes[0].Related[0].Relationship != "provider_recommendation_not_concept_evidence" ||
		len(b.Outcomes[0].Candidates) != 1 || b.Outcomes[0].Candidates[0].ExternalIDs["tmdb"] != "44" || requests.Load() != 3 {
		t.Fatalf("related source ID workflow incorrect %#v %v", b, e)
	}
}

func TestJellyfinModernAuthorizationExactIDAndRelatedWithoutWrites(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != "GET" || r.Header.Get("Authorization") != "MediaBrowser Token="+strconv.Quote(strings.Repeat("K", 32)) {
			t.Errorf("Jellyfin GET-only / modern authentication violated: %s", r.Method)
		}
		switch r.URL.Path {
		case "/Items":
			if r.URL.Query().Get("SearchTerm") != "Collision" || r.URL.Query().Get("Fields") != "ProviderIds,ProductionYear" ||
				r.URL.Query().Get("Limit") != "10" {
				t.Errorf("wrong bounded local search %s", r.URL)
			}
			_, _ = w.Write([]byte(`{"Items":[
				{"Id":"aaaaaaaaaaaa","Name":"Collision","Type":"Movie","ProductionYear":2020,"ProviderIds":{"Tmdb":"123"}},
				{"Id":"bbbbbbbbbbbb","Name":"Collision","Type":"Movie","ProductionYear":2020,"ProviderIds":{"Tmdb":"456"}},
				{"Id":"dddddddddddd","Name":"Collision","Type":"Movie","ProductionYear":2020,"ProviderIds":{}}
			],"TotalRecordCount":3}`))
		case "/Items/aaaaaaaaaaaa":
			_, _ = w.Write([]byte(`{"Id":"aaaaaaaaaaaa","Name":"Collision","Type":"Movie","ProductionYear":2020,"ProviderIds":{"Tmdb":"123"}}`))
		case "/Items/aaaaaaaaaaaa/Similar":
			if r.URL.Query().Get("Limit") != "8" {
				t.Error("unbounded related local items")
			}
			_, _ = w.Write([]byte(`{"Items":[{"Id":"cccccccccccc","Name":"Related Fictional","Type":"Movie","ProductionYear":2021,"ProviderIds":{"Imdb":"tt1234567"}}],"TotalRecordCount":1}`))
		default:
			t.Errorf("unexpected local path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("BCE_JELLYFIN_BASE_URL", srv.URL)
	t.Setenv("BCE_JELLYFIN_TOKEN", strings.Repeat("K", 32))
	a, e := lookupMedia(context.Background(), MediaLookupRequest{Title: "Collision", Kind: "movie", TMDbID: "123", Providers: []string{"jellyfin"}, IncludeRelated: true})
	if e != nil || requests.Load() != 1 || len(a.Outcomes[0].Candidates) != 1 ||
		a.Outcomes[0].Candidates[0].ItemID != "aaaaaaaaaaaa" ||
		a.Outcomes[0].Candidates[0].MatchBasis != "exact_local_tmdb_provider_id" ||
		len(a.Outcomes[0].Related) != 0 || len(a.Outcomes[0].Warnings) < 1 {
		t.Fatalf("wrong identity-first collision disposition %#v %v", a, e)
	}
	b, e := lookupMedia(context.Background(), MediaLookupRequest{Kind: "movie", JellyfinItemID: "aaaaaaaaaaaa", Providers: []string{"jellyfin"}, IncludeRelated: true})
	if e != nil || len(b.Outcomes[0].Candidates) != 1 || len(b.Outcomes[0].Related) != 1 ||
		b.Outcomes[0].Related[0].Relationship != "jellyfin_similar_not_cpe_evidence" || requests.Load() != 3 {
		t.Fatalf("Jellyfin similar lookup incorrect %#v %v", b, e)
	}
}

func TestJellyfinNoCredentialsAndUnavailableAreNotOwnedOrAbsent(t *testing.T) {
	t.Setenv("BCE_JELLYFIN_BASE_URL", "")
	t.Setenv("BCE_JELLYFIN_TOKEN", "")
	o, e := lookupMedia(context.Background(), MediaLookupRequest{Kind: "movie", Title: "Fictional", Providers: []string{"jellyfin"}})
	if e != nil || o.Outcomes[0].State != "unconfigured_local_library" || len(o.Outcomes[0].Candidates) != 0 {
		t.Fatalf("unconfigured claimed local holdings %#v %v", o, e)
	}
}

func TestMediaLookupHTTPRejectsUntrustedURLs(t *testing.T) {
	mux := http.NewServeMux()
	registerMediaLookup(mux)
	payload := []byte(`{"kind":"movie","title":"sample","providers":["jellyfin"],"url":"http://169.254.169.254"}`)
	r := httptest.NewRequest("POST", "/api/v1/media/lookup", bytes.NewReader(payload))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("unexpected URL accepted: %d %s", w.Code, w.Body.String())
	}
	allowed := []byte(`{"kind":"movie","title":"sample","providers":["imdb","tmdb"],"allow_external_lookup":false}`)
	r = httptest.NewRequest("POST", "/api/v1/media/lookup", bytes.NewReader(allowed))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("allowlisted query rejected: %d", w.Code)
	}
	var result MediaLookupResponse
	if e := json.Unmarshal(w.Body.Bytes(), &result); e != nil || result.Outcomes[1].State != "requires_explicit_external_consent" {
		t.Fatalf("unexpected optional provider behavior %#v %v", result, e)
	}
}
