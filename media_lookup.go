package main

// Bounded, explicitly requested metadata retrieval. This is not a second media
// identity authority, user behavioral history, crawler, recommendation engine,
// acquisition path, or CPE concept adjudicator.
import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/mark-e-deyoung/bitmagnet-concept-explorer/internal/contracts"
)

type MediaLookupRequest struct {
	Language            string   `json:"language,omitempty"`
	Title               string   `json:"title,omitempty"`
	Kind                string   `json:"kind"`
	Year                int      `json:"year,omitempty"`
	IMDbID              string   `json:"imdb_id,omitempty"`
	TMDbID              string   `json:"tmdb_id,omitempty"`
	JellyfinItemID      string   `json:"jellyfin_item_id,omitempty"`
	Providers           []string `json:"providers"`
	IncludeRelated      bool     `json:"include_related,omitempty"`
	AllowExternalLookup bool     `json:"allow_external_lookup,omitempty"`
	TMDbGenreIDs        []int    `json:"tmdb_genre_ids,omitempty"`
	TMDbKeywordIDs      []int    `json:"tmdb_keyword_ids,omitempty"`
	JellyfinGenres      []string `json:"jellyfin_genres,omitempty"`
}

type MediaIdentityCandidate struct {
	Source       string            `json:"source"`
	ItemID       string            `json:"item_id"`
	Kind         string            `json:"kind"`
	Title        string            `json:"title"`
	Year         int               `json:"year,omitempty"`
	ExternalIDs  map[string]string `json:"external_ids,omitempty"`
	MatchBasis   string            `json:"match_basis"`
	Relationship string            `json:"relationship"`
	GenreIDs     []int             `json:"source_genre_ids,omitempty"`
}

type MediaProviderOutcome struct {
	Source      string                   `json:"source"`
	State       string                   `json:"state"`
	Candidates  []MediaIdentityCandidate `json:"candidates"`
	Related     []MediaIdentityCandidate `json:"related"`
	Warnings    []string                 `json:"warnings"`
	Attribution string                   `json:"attribution,omitempty"`
}

type MediaLookupResponse struct {
	Version  string                 `json:"version"`
	ReadOnly bool                   `json:"read_only"`
	Scope    string                 `json:"scope"`
	Outcomes []MediaProviderOutcome `json:"outcomes"`
	Warnings []string               `json:"warnings"`
}

var (
	imdbIDPattern     = regexp.MustCompile(`^tt[0-9]{7,10}$`)
	jellyfinIDPattern = regexp.MustCompile(`^[0-9a-fA-F-]{8,64}$`)
	tmdbIDPattern     = regexp.MustCompile(`^[1-9][0-9]{0,10}$`)
)

func validateMediaLookup(q MediaLookupRequest) error {
	if q.Kind != "movie" && q.Kind != "series" {
		return errors.New("kind must be movie or series")
	}
	if q.Language != "" && !regexp.MustCompile("^[a-z]{2}(-[A-Z]{2})?$").MatchString(q.Language) {
		return errors.New("invalid provider language")
	}
	if len([]rune(q.Title)) > 160 {
		return errors.New("invalid title")
	}
	for _, c := range q.Title {
		if c == 0 || c == 10 || c == 13 {
			return errors.New("invalid title control character")
		}
	}
	nativeConcepts := len(q.TMDbGenreIDs)+len(q.TMDbKeywordIDs)+len(q.JellyfinGenres)
	if nativeConcepts > 6 || len(q.TMDbGenreIDs) > 3 || len(q.TMDbKeywordIDs) > 3 || len(q.JellyfinGenres) > 3 {
		return errors.New("provider-native concept filter budget exceeded")
	}
	for _, id := range append(append([]int{},q.TMDbGenreIDs...),q.TMDbKeywordIDs...) {
		if id < 1 || id > 1000000000 {
			return errors.New("invalid provider-native numeric concept identifier")
		}
	}
	for _, value := range q.JellyfinGenres {
		if len([]rune(value)) < 2 || len([]rune(value)) > 50 || strings.TrimSpace(value) != value ||
			strings.ContainsAny(value, "/?&#") {
			return errors.New("invalid local source-native genre name")
		}
		for _, char := range value {
			if char == 0 || char == 10 || char == 13 {
				return errors.New("invalid local source-native genre control")
			}
		}
	}
	if nativeConcepts > 0 && (strings.TrimSpace(q.Title)!="" || q.IMDbID!="" || q.TMDbID!="" || q.JellyfinItemID!="" || q.IncludeRelated || q.Year!=0) {
		return errors.New("provider-native concept discovery is exclusive of item identity/related lookup")
	}
	if strings.TrimSpace(q.Title) == "" && q.IMDbID == "" && q.TMDbID == "" && q.JellyfinItemID == "" && nativeConcepts==0 {
		return errors.New("title, explicit media identity or provider-native concept filter required")
	}
	if q.Year != 0 && (q.Year < 1888 || q.Year > 2100) {
		return errors.New("year outside bounded historical range")
	}
	if q.IMDbID != "" && !imdbIDPattern.MatchString(q.IMDbID) {
		return errors.New("invalid IMDb ID")
	}
	if q.TMDbID != "" && !tmdbIDPattern.MatchString(q.TMDbID) {
		return errors.New("invalid TMDb ID")
	}
	if q.JellyfinItemID != "" && !jellyfinIDPattern.MatchString(q.JellyfinItemID) {
		return errors.New("invalid Jellyfin item ID")
	}
	if len(q.Providers) == 0 || len(q.Providers) > 3 {
		return errors.New("1-3 explicit source providers required")
	}
	seen := map[string]bool{}
	for _, s := range q.Providers {
		if s != "tmdb" && s != "jellyfin" && s != "imdb" {
			return errors.New("unsupported media metadata source")
		}
		if seen[s] {
			return errors.New("duplicate media source")
		}
		seen[s] = true
	}
	return nil
}

func mediaLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func providerURL(raw string, remoteTMDb bool) (*url.URL, error) {
	u, e := url.Parse(raw)
	if e != nil || u.Host == "" || u.User != nil || u.Fragment != "" || u.RawQuery != "" || u.RawPath != "" ||
		(u.Scheme != "https" && u.Scheme != "http") {
		return nil, errors.New("invalid configured media provider origin")
	}
	if remoteTMDb {
		if u.Scheme == "https" && strings.EqualFold(u.Hostname(), "api.themoviedb.org") && u.Path == "/3" && u.Port() == "" {
			return u, nil
		}
		if u.Scheme == "http" && u.Path == "/3" && mediaLoopbackHost(u.Hostname()) {
			return u, nil // explicit test-only loopback provider, never a public arbitrary host
		}
		return nil, errors.New("TMDb endpoint must be official HTTPS or loopback fixture")
	}
	if u.Path != "" && u.Path != "/" {
		return nil, errors.New("Jellyfin endpoint must have no path")
	}
	host := u.Hostname()
	if !mediaLoopbackHost(host) {
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsPrivate() {
			return nil, errors.New("Jellyfin endpoint must be a literal private/loopback address")
		}
	}
	return u, nil
}

func mediaHTTPClient() *http.Client {
	return &http.Client{
		Timeout:       7 * time.Second,
		Transport:     &http.Transport{Proxy: nil},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// Fetch only bounded JSON from a fixed server-configured endpoint, with the
// server-held token in a header. No raw Bitmagnet items or API keys in URLs.
func readMediaJSON(ctx context.Context, client *http.Client, target, authHeader, token string, dest any) (string, error) {
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if e != nil {
		return "invalid_request", e
	}
	req.Header.Set(authHeader, token)
	req.Header.Set("Accept", "application/json")
	res, e := client.Do(req)
	if e != nil {
		return "unavailable", errors.New("media provider request unavailable")
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case 200:
	case 401, 403:
		return "authentication_refused", errors.New("media provider rejected credentials")
	case 429:
		return "rate_limited_no_retry", errors.New("media provider rate limited request")
	case 404:
		return "not_found_non_diagnostic", nil
	default:
		return "provider_failure", errors.New("media provider returned an unsuccessful status")
	}
	dec := json.NewDecoder(io.LimitReader(res.Body, (1<<20)+1))
	if e := dec.Decode(dest); e != nil {
		return "invalid_source_response", errors.New("malformed media provider JSON")
	}
	if e := dec.Decode(new(any)); e != io.EOF {
		return "invalid_source_response", errors.New("trailing or oversized media provider JSON")
	}
	return "ok", nil
}

func emptyOutcome(source string) MediaProviderOutcome {
	return MediaProviderOutcome{
		Source: source, State: "unavailable",
		Candidates: []MediaIdentityCandidate{}, Related: []MediaIdentityCandidate{},
		Warnings: []string{},
	}
}

func lookupMedia(ctx context.Context, q MediaLookupRequest) (MediaLookupResponse, error) {
	if err := validateMediaLookup(q); err != nil {
		return MediaLookupResponse{}, err
	}
	out := MediaLookupResponse{
		Version: "media-lookup-provisional-v1", ReadOnly: true,
		Scope:    "explicit_identity_or_bounded_first_page_only",
		Outcomes: []MediaProviderOutcome{},
		Warnings: []string{
			"Media title/genre/recommendation metadata is not accepted CPE concept evidence",
			"Provider-native genre/keyword IDs and local genre names require independently qualified CPE source mapping before any semantic claim",
			"Title/year matches are candidates, not resolved ownership or proof of absence",
			"Third-party content is never automatically forwarded to models or persisted",
		},
	}
	client := mediaHTTPClient()
	for _, source := range q.Providers {
		var p MediaProviderOutcome
		switch source {
		case "tmdb":
			p = lookupTMDb(ctx, client, q)
		case "jellyfin":
			p = lookupJellyfin(ctx, client, q)
		case "imdb":
			p = emptyOutcome("imdb")
			p.State = "licensed_source_not_configured"
			p.Warnings = append(p.Warnings, "No IMDb scraping/API bypass: use licensed IMDb access or the allowed IMDb-ID crosswalk through TMDb")
			p.Attribution = "IMDb is a third-party source; licensing must be established separately"
		}
		out.Outcomes = append(out.Outcomes, p)
	}
	return out, nil
}

func registerMediaLookup(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/media/lookup", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, contracts.QueryJSONBytes)
		defer r.Body.Close()
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		var in MediaLookupRequest
		if e := dec.Decode(&in); e != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid media lookup request"})
			return
		}
		if e := dec.Decode(new(any)); e != io.EOF {
			writeJSON(w, 400, map[string]string{"error": "trailing media lookup request"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 18*time.Second)
		defer cancel()
		out, e := lookupMedia(ctx, in)
		if e != nil {
			writeJSON(w, 422, map[string]string{"error": e.Error()})
			return
		}
		writeJSON(w, 200, out)
	})
}
