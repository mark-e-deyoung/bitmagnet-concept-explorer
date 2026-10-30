package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Bounded source-wide admission protects the local SearXNG service across clients.
var globalSearxngSlots = make(chan struct{}, 2)

// Rate-admission and refusal cooldown are shared across all callers. There is
// no retry on provider refusal. This is a minimum local safeguard, not a
// substitute for each engine's own terms, quotas and SearXNG configuration.
var errSearXNGBackoff = errors.New("SearXNG denied or cooldown active")
var searXNGPace = struct {
	sync.Mutex
	Next         map[string]time.Time
	BlockedUntil map[string]time.Time
	Recent       map[string][]time.Time
}{Next: map[string]time.Time{}, BlockedUntil: map[string]time.Time{}, Recent: map[string][]time.Time{}}

func paceSearXNG(ctx context.Context, endpoint string) error {
	searXNGPace.Lock()
	now := time.Now()
	if until := searXNGPace.BlockedUntil[endpoint]; now.Before(until) {
		searXNGPace.Unlock()
		return fmt.Errorf("%w: endpoint cooling down after refusal", errSearXNGBackoff)
	}
	// SearXNG's documented default automated JSON API limiter is four per
	// rolling hour. Our conservative process ceiling is THREE per hour.
	recent := searXNGPace.Recent[endpoint]
	pruned := recent[:0]
	for _, t := range recent {
		if now.Sub(t) < time.Hour {
			pruned = append(pruned, t)
		}
	}
	searXNGPace.Recent[endpoint] = pruned
	if len(pruned) >= 3 {
		searXNGPace.Unlock()
		return fmt.Errorf("%w: local SearXNG JSON hourly request budget exhausted", errSearXNGBackoff)
	}
	start := now
	if next := searXNGPace.Next[endpoint]; next.After(start) {
		start = next
	}
	searXNGPace.Recent[endpoint] = append(searXNGPace.Recent[endpoint], start)
	searXNGPace.Next[endpoint] = start.Add(2 * time.Second)
	searXNGPace.Unlock()
	if delay := time.Until(start); delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			return fmt.Errorf("%w: pacing canceled", errSearXNGBackoff)
		}
	}
	return nil
}
func searXNGBackoff(endpoint string, status int, retryAfter string) {
	var min time.Duration
	switch status {
	case 403:
		min = time.Hour
	case 429:
		min = 10 * time.Minute
	case 503:
		min = 2 * time.Minute
	default:
		return
	}
	if n, e := strconv.Atoi(strings.TrimSpace(retryAfter)); e == nil && n > 0 {
		if d := time.Duration(n) * time.Second; d > min {
			min = d
		}
	} else if t, e := http.ParseTime(retryAfter); e == nil {
		if d := time.Until(t); d > min {
			min = d
		}
	}
	if min > 24*time.Hour {
		min = 24 * time.Hour
	}
	searXNGPace.Lock()
	until := time.Now().Add(min)
	if until.After(searXNGPace.BlockedUntil[endpoint]) {
		searXNGPace.BlockedUntil[endpoint] = until
	}
	searXNGPace.Unlock()
}

// ExternalObservation is never equated with a Bitmagnet infohash or CPE assertion.
type ExternalObservation struct {
	Title   string   `json:"title"`
	URL     string   `json:"url"`
	Engines []string `json:"engines,omitempty"`
	Source  string   `json:"source"`
}

func searXNGEndpoint() (string, error) {
	raw := strings.TrimSpace(os.Getenv("BCE_SEARXNG_URL"))
	if raw == "" {
		return "", errors.New("SearXNG is not configured")
	}
	u, e := url.Parse(raw)
	if e != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "/search" {
		return "", errors.New("invalid configured SearXNG endpoint: require fixed http(s) /search URL")
	}
	return raw, nil
}

type SearXNGResearchEnvelope struct {
	Hits        []ExternalObservation
	Suggestions []string
}

func searchSearXNG(ctx context.Context, query string) ([]ExternalObservation, error) {
	return searchSearXNGScoped(ctx, query, "")
}
func searchSearXNGScoped(ctx context.Context, query, language string) ([]ExternalObservation, error) {
	out, e := searchSearXNGDetailed(ctx, query, language)
	return out.Hits, e
}
func searchSearXNGDetailed(ctx context.Context, query, language string) (SearXNGResearchEnvelope, error) {
	endpoint, e := searXNGEndpoint()
	if e != nil {
		return SearXNGResearchEnvelope{}, e
	}
	select {
	case globalSearxngSlots <- struct{}{}:
		defer func() { <-globalSearxngSlots }()
	case <-ctx.Done():
		return SearXNGResearchEnvelope{}, errors.New("SearXNG admission cancelled")
	}
	if e := paceSearXNG(ctx, endpoint); e != nil {
		return SearXNGResearchEnvelope{}, e
	}
	if language != "" && language != "und" && !sweepLanguage(language) {
		return SearXNGResearchEnvelope{}, errors.New("invalid requested SearXNG language")
	}
	if !safeExternalSweepTerm(query) {
		return SearXNGResearchEnvelope{}, errors.New("unsafe external query syntax")
	}

	u, _ := url.Parse(endpoint)
	q := u.Query()
	q.Set("q", query)
	q.Set("format", "json")
	q.Set("pageno", "1")
	if language != "" && language != "und" {
		q.Set("language", language)
	}
	u.RawQuery = q.Encode()
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if e != nil {
		return SearXNGResearchEnvelope{}, e
	}
	client := &http.Client{
		Timeout:       8 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		Transport:     &http.Transport{Proxy: nil, MaxIdleConnsPerHost: 2, IdleConnTimeout: 5 * time.Second},
	}
	resp, e := client.Do(req)
	if e != nil {
		return SearXNGResearchEnvelope{}, errors.New("SearXNG request failed or timed out")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		searXNGBackoff(endpoint, resp.StatusCode, resp.Header.Get("Retry-After"))
		if resp.StatusCode == 403 || resp.StatusCode == 429 || resp.StatusCode == 503 {
			return SearXNGResearchEnvelope{}, fmt.Errorf("%w: HTTP %d; no retry", errSearXNGBackoff, resp.StatusCode)
		}
		return SearXNGResearchEnvelope{}, fmt.Errorf("SearXNG returned HTTP %d (JSON may be disabled)", resp.StatusCode)
	}
	var payload struct {
		Suggestions []json.RawMessage `json:"suggestions"`
		Results     []struct {
			Title   string   `json:"title"`
			URL     string   `json:"url"`
			Engine  string   `json:"engine"`
			Engines []string `json:"engines"`
		} `json:"results"`
	}
	if e := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&payload); e != nil {
		return SearXNGResearchEnvelope{}, errors.New("invalid SearXNG JSON")
	}
	out := make([]ExternalObservation, 0, 20)
	for _, item := range payload.Results {
		if len(out) >= 20 {
			break
		}
		u, e := url.Parse(item.URL)
		if e != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil {
			continue
		}
		title := strings.TrimSpace(item.Title)
		if title == "" {
			continue
		}
		if len([]rune(title)) > 256 {
			title = string([]rune(title)[:256])
		}
		engines := append([]string(nil), item.Engines...)
		if len(engines) == 0 && item.Engine != "" {
			engines = []string{item.Engine}
		}
		if len(engines) > 8 {
			engines = engines[:8]
		}
		out = append(out, ExternalObservation{Title: title, URL: u.String(), Engines: engines, Source: "searxng"})
	}
	hints := []string{}
	seenHints := map[string]bool{}
	for _, raw := range payload.Suggestions {
		var hint string
		if json.Unmarshal(raw, &hint) != nil {
			continue
		}
		hint = strings.TrimSpace(hint)
		key := strings.ToLower(hint)
		if key == strings.ToLower(strings.TrimSpace(query)) || seenHints[key] || !safeExternalSweepTerm(hint) {
			continue
		}
		if len(hints) >= 8 {
			break
		}
		hints = append(hints, hint)
		seenHints[key] = true
	}
	return SearXNGResearchEnvelope{Hits: out, Suggestions: hints}, nil
}
