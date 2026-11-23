package main

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type jellyfinItem struct {
	ID             string            `json:"Id"`
	Name           string            `json:"Name"`
	Type           string            `json:"Type"`
	ProductionYear int               `json:"ProductionYear"`
	ProviderIDs    map[string]string `json:"ProviderIds"`
}
type jellyfinItems struct {
	Items            []jellyfinItem `json:"Items"`
	TotalRecordCount int            `json:"TotalRecordCount"`
}

func jellyfinCandidate(r jellyfinItem, relationship string, basis string) MediaIdentityCandidate {
	kind := "movie"
	if r.Type == "Series" {
		kind = "series"
	}
	ids := map[string]string{}
	for key, value := range r.ProviderIDs {
		switch strings.ToLower(key) {
		case "tmdb", "imdb", "tvdb":
			if len(value) <= 64 {
				ids[strings.ToLower(key)] = value
			}
		}
	}
	return MediaIdentityCandidate{
		Source: "jellyfin", ItemID: r.ID, Kind: kind, Title: r.Name,
		Year: r.ProductionYear, ExternalIDs: ids,
		MatchBasis: basis, Relationship: relationship,
	}
}

func lookupJellyfin(ctx context.Context, client *http.Client, q MediaLookupRequest) MediaProviderOutcome {
	p := emptyOutcome("jellyfin")
	base := os.Getenv("BCE_JELLYFIN_BASE_URL")
	token := os.Getenv("BCE_JELLYFIN_TOKEN")
	if base == "" || token == "" {
		p.State = "unconfigured_local_library"
		return p
	}
	if len(token) > 256 || len(token) < 16 {
		p.State = "invalid_server_token_configuration"
		return p
	}
	for _, b := range token {
		if b == 34 || b == 92 || b == 10 || b == 13 {
			p.State = "invalid_server_token_configuration"
			return p
		}
	}
	baseURL, err := providerURL(base, false)
	if err != nil {
		p.State = "invalid_server_provider_configuration"
		return p
	}
	endpoint := strings.TrimSuffix(baseURL.String(), "/")
	auth := "MediaBrowser Token=" + strconv.Quote(token)
	var items jellyfinItems
	var direct bool
	switch {
	case len(q.JellyfinGenres) > 0:
		params := url.Values{}
		params.Set("Genres", strings.Join(q.JellyfinGenres, ","))
		params.Set("IncludeItemTypes", "Movie,Series")
		params.Set("Fields", "ProviderIds,ProductionYear")
		params.Set("Recursive", "true")
		params.Set("Limit", "10")
		state, err := readMediaJSON(ctx, client, endpoint+"/Items?"+params.Encode(), "Authorization", auth, &items)
		if err != nil || state != "ok" {
			p.State = state
			return p
		}
	case q.JellyfinItemID != "":
		var item jellyfinItem
		state, err := readMediaJSON(ctx, client, endpoint+"/Items/"+q.JellyfinItemID, "Authorization", auth, &item)
		if err != nil || state != "ok" {
			p.State = state
			return p
		}
		items.Items = []jellyfinItem{item}
		direct = true
	case strings.TrimSpace(q.Title) != "":
		params := url.Values{}
		params.Set("SearchTerm", strings.TrimSpace(q.Title))
		params.Set("IncludeItemTypes", "Movie,Series")
		params.Set("Fields", "ProviderIds,ProductionYear")
		params.Set("Recursive", "true")
		params.Set("Limit", "10")
		state, err := readMediaJSON(ctx, client, endpoint+"/Items?"+params.Encode(), "Authorization", auth, &items)
		if err != nil || state != "ok" {
			p.State = state
			return p
		}
	default:
		p.State = "title_or_confirmed_local_item_required"
		return p
	}
	if len(items.Items) > 10 {
		p.State = "invalid_source_response"
		return p
	}
	if items.TotalRecordCount > len(items.Items) {
		p.Warnings = append(p.Warnings, "Local library result page is capped; missing items do not establish absence")
	}
	for _, item := range items.Items {
		if item.Type != "Movie" && item.Type != "Series" {
			continue
		}
		if (q.Kind == "movie" && item.Type != "Movie") || (q.Kind == "series" && item.Type != "Series") {
			continue
		}
		if item.ID == "" {
			continue
		}
		basis := "title_search_candidate_unconfirmed"
		if len(q.JellyfinGenres) > 0 {
			basis = "source_native_genre_candidate_unmapped"
		}
		if direct {
			basis = "selected_local_item_detail_unconfirmed"
		}
		mismatch := false
		if q.TMDbID != "" {
			existing := ""
			for k, v := range item.ProviderIDs {
				if strings.EqualFold(k, "tmdb") {
					existing = v
					break
				}
			}
			if existing != q.TMDbID {
				// An absent provider ID cannot establish ownership from a title.
				mismatch = true
			}
			if existing == q.TMDbID {
				basis = "exact_local_tmdb_provider_id"
			}
		}
		if q.IMDbID != "" {
			existing := ""
			for k, v := range item.ProviderIDs {
				if strings.EqualFold(k, "imdb") {
					existing = v
					break
				}
			}
			if existing != q.IMDbID {
				// Do not silently accept a same-title item when a known ID is missing.
				mismatch = true
			}
			if existing == q.IMDbID {
				basis = "exact_local_imdb_provider_id"
			}
		}
		if mismatch {
			p.Warnings = append(p.Warnings, "Local item had conflicting external identity; review required")
			continue
		}
		if q.Year > 0 && item.ProductionYear != 0 && item.ProductionYear != q.Year && basis == "title_search_candidate_unconfirmed" {
			p.Warnings = append(p.Warnings, "Same title with conflicting production year is not automatically owned")
			continue
		}
		p.Candidates = append(p.Candidates, jellyfinCandidate(item, "candidate", basis))
	}
	p.State = "candidates_need_identity_confirmation"
	if len(q.JellyfinGenres) > 0 {
		p.State = "provider_native_concept_results_unmapped"
		p.Warnings = append(p.Warnings, "Local Jellyfin genre filter is source-native, not an accepted CPE concept mapping or full-library coverage")
	}
	if len(p.Candidates) == 0 {
		p.State = "no_candidate_non_diagnostic"
	}
	if q.IncludeRelated {
		if q.JellyfinItemID == "" {
			p.Warnings = append(p.Warnings, "Related local titles require a selected Jellyfin item ID; never choose the first title match")
		} else {
			params := url.Values{}
			params.Set("Limit", "8")
			params.Set("Fields", "ProviderIds,ProductionYear")
			var similar jellyfinItems
			target := endpoint + "/Items/" + q.JellyfinItemID + "/Similar?" + params.Encode()
			state, err := readMediaJSON(ctx, client, target, "Authorization", auth, &similar)
			if err != nil || state != "ok" {
				p.Warnings = append(p.Warnings, "Local related-item lookup "+state+"; no absence inference")
			} else {
				for _, item := range similar.Items {
					if item.ID == "" || len(p.Related) == 8 {
						break
					}
					if item.Type != "Movie" && item.Type != "Series" {
						continue
					}
					p.Related = append(p.Related, jellyfinCandidate(item, "jellyfin_similar_not_cpe_evidence", "local_provider_related_only"))
				}
				if similar.TotalRecordCount > 8 || len(similar.Items) > 8 {
					p.Warnings = append(p.Warnings, "Related local items bounded")
				}
			}
		}
	}
	return p
}
