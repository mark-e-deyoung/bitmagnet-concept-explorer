package main

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type tmdbRecord struct {
	ID int `json:"id"`
	Title string `json:"title"`
	Name string `json:"name"`
	ReleaseDate string `json:"release_date"`
	FirstAirDate string `json:"first_air_date"`
	GenreIDs []int `json:"genre_ids"`
}
type tmdbList struct {
	Results []tmdbRecord `json:"results"`
	TotalPages int `json:"total_pages"`
}
type tmdbFind struct {
	MovieResults []tmdbRecord `json:"movie_results"`
	TVResults []tmdbRecord `json:"tv_results"`
}

const tmdbAttribution = "This product uses the TMDB API but is not endorsed or certified by TMDB."

func tmdbCandidate(r tmdbRecord, kind, relationship, basis string) MediaIdentityCandidate {
	title, date := r.Title, r.ReleaseDate
	if kind == "series" {
		title, date = r.Name, r.FirstAirDate
	}
	year := 0
	if len(date) >= 4 {
		for _, c := range date[:4] {
			if c < '0' || c > '9' { year = 0; break }
			year = year*10 + int(c-'0')
		}
	}
	return MediaIdentityCandidate{
		Source: "tmdb", ItemID: mediaPositiveID(r.ID), Kind: kind, Title: title, Year: year,
		ExternalIDs: map[string]string{"tmdb": mediaPositiveID(r.ID)},
		MatchBasis: basis, Relationship: relationship, GenreIDs: r.GenreIDs,
	}
}

func mediaPositiveID(id int) string {
	if id <= 0 { return "" }
	return strconv.Itoa(id)
}

func lookupTMDb(ctx context.Context, client *http.Client, q MediaLookupRequest) MediaProviderOutcome {
	p := emptyOutcome("tmdb")
	p.Attribution = tmdbAttribution
	if !q.AllowExternalLookup {
		p.State = "requires_explicit_external_consent"
		return p
	}
	token := os.Getenv("BCE_TMDB_READ_TOKEN")
	if token == "" {
		p.State = "unconfigured_read_token"
		return p
	}
	base := os.Getenv("BCE_TMDB_API_BASE")
	if base == "" { base = "https://api.themoviedb.org/3" }
	origin, err := providerURL(base, true)
	if err != nil {
		p.State = "invalid_server_provider_configuration"
		return p
	}
	endpoint := strings.TrimSuffix(origin.String(), "/")
	mode := q.Kind
	if mode == "series" { mode = "tv" }
	path := ""
	params := url.Values{}
	switch {
	case q.TMDbID != "":
		path = "/" + mode + "/" + q.TMDbID
	case q.IMDbID != "":
		path = "/find/" + q.IMDbID
		params.Set("external_source", "imdb_id")
	case strings.TrimSpace(q.Title) != "":
		path = "/search/" + mode
		params.Set("query", strings.TrimSpace(q.Title))
		if q.Year > 0 {
			if mode == "movie" { params.Set("year", strconv.Itoa(q.Year)) } else { params.Set("first_air_date_year", strconv.Itoa(q.Year)) }
		}
	default:
		p.State = "insufficient_identity_non_diagnostic"
		return p
	}
	if q.Language != "" { params.Set("language", q.Language) }
	full := endpoint+path
	if len(params)>0 { full += "?"+params.Encode() }
	var records []tmdbRecord
	var pages int
	if q.TMDbID != "" {
		var one tmdbRecord
		state, err := readMediaJSON(ctx, client, full, "Authorization", "Bearer "+token, &one)
		if err != nil || state != "ok" {
			p.State = state
			return p
		}
		if one.ID > 0 { records = []tmdbRecord{one} }
	} else if q.IMDbID != "" {
		var found tmdbFind
		state, err := readMediaJSON(ctx, client, full, "Authorization", "Bearer "+token, &found)
		if err != nil || state != "ok" {
			p.State = state
			return p
		}
		if q.Kind == "movie" { records = found.MovieResults } else { records = found.TVResults }
	} else {
		var list tmdbList
		state, err := readMediaJSON(ctx, client, full, "Authorization", "Bearer "+token, &list)
		if err != nil || state != "ok" {
			p.State = state
			return p
		}
		records = list.Results
		pages = list.TotalPages
	}
	if len(records) > 20 {
		p.State = "invalid_source_response"
		return p
	}
	if pages > 1 { p.Warnings = append(p.Warnings, "Only first provider page was observed, not an exhaustive search") }
	for _, record := range records {
		if record.ID < 1 || len(p.Candidates) == 10 { break }
		basis := "title_candidate_unconfirmed"
		if q.TMDbID != "" { basis = "supplied_tmdb_id_detail_unconfirmed" }
		if q.IMDbID != "" { basis = "provider_imdb_id_crosswalk" }
		p.Candidates = append(p.Candidates, tmdbCandidate(record, q.Kind, "candidate", basis))
	}
	if len(records)>len(p.Candidates) { p.Warnings = append(p.Warnings, "Candidate list is bounded; no match absence inference") }
	p.State = "candidates_need_identity_confirmation"
	if len(p.Candidates)==0 { p.State = "no_candidate_non_diagnostic" }
	if q.IncludeRelated {
		if q.TMDbID == "" {
			p.Warnings = append(p.Warnings, "Related-title lookup requires operator-confirmed TMDb ID, not title-only or inferred lookup")
		} else {
			var similar tmdbList
			params := url.Values{}
			if q.Language!="" { params.Set("language",q.Language) }
			target := endpoint+"/"+mode+"/"+q.TMDbID+"/recommendations"
			if len(params)>0 {target += "?"+params.Encode()}
			state, err := readMediaJSON(ctx, client, target, "Authorization", "Bearer "+token, &similar)
			if err != nil || state!="ok" {
				p.Warnings=append(p.Warnings,"Related-title provider request "+state+"; no absence inference")
			} else {
				for _, v := range similar.Results {
					if v.ID<=0 || len(p.Related)==8 {break}
					p.Related=append(p.Related,tmdbCandidate(v,q.Kind,"provider_recommendation_not_concept_evidence","provider_related_only"))
				}
				if len(similar.Results)>8 || similar.TotalPages>1 {p.Warnings=append(p.Warnings,"Related titles bounded to first provider page")}
			}
		}
	}
	return p
}
