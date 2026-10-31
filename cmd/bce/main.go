// Command bce is a thin, non-mutating automation/agent client for Concept Explorer.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/mark-e-deyoung/bitmagnet-concept-explorer/internal/contracts"
	"github.com/mark-e-deyoung/bitmagnet-concept-explorer/internal/deadline"
)

const maxCLIResponseBytes = 4 << 20

// Keep the client open beyond the server's bounded sweep deadline and write
// allowance so callers receive the final full/partial response when available.
func main() {
	service := flag.String("server", "http://127.0.0.1:3086", "Concept Explorer origin")
	query := flag.String("query", "", "Literal query for plan/execute")
	expansions := flag.String("expand", "", "Comma-separated exploratory terms")
	operation := flag.String("operation", "capabilities", "capabilities, limits, concepts, concept, plan, execute, facets, observed-graph, cpe-preflight, infer, sweep-plan, sweep, discovery-preview or research-evidence")
	languages := flag.String("languages", "", "Comma-separated observed language IDs")
	media := flag.String("media", "", "Native Bitmagnet media type")
	minSeeders := flag.Int("min-seeders", -1, "Minimum observed seeders (-1 unset)")
	minLeechers := flag.Int("min-leechers", -1, "Minimum observed leechers (-1 unset)")
	adult := flag.String("adult-metadata", "any", "any, yes, no or unknown (not a policy verdict)")
	sortBy := flag.String("sort", "info_hash", "seeders_desc, leechers_desc, title_asc etc")
	groupBy := flag.String("group", "none", "none, media_type, language, adult_metadata or track")
	useWeb := flag.Bool("web", false, "Explicitly send literal query to configured SearXNG")
	conceptID := flag.String("concept-id", "", "Exact concept ID for inspect operation")
	researchSection := flag.String("research-section", "", "Optional public research evidence section: works, entities, claims, art_candidates, source_inventory")
	selectedConcepts := flag.String("concept-ids", "", "Comma-separated exact IDs from pinned provisional graph, opt-in")
	facetGroups := flag.String("facet-groups", "contentType,language,genre", "Comma-separated native groups for -operation facets (1-4)")
	inputJSON := flag.String("input", "-", "JSON input file or - for stdin for observed-graph, cpe-preflight, infer, sweep-plan, sweep and discovery-preview")
	flag.Parse()
	serviceURL, urlErr := validateServiceOrigin(*service)
	if urlErr != nil {
		fmt.Fprintln(os.Stderr, urlErr)
		os.Exit(2)
	}
	authUser, authPassword := os.Getenv("BCE_AUTH_USER"), os.Getenv("BCE_AUTH_PASSWORD")
	if (authUser == "") != (authPassword == "") {
		fmt.Fprintln(os.Stderr, "BCE_AUTH_USER and BCE_AUTH_PASSWORD must be configured together")
		os.Exit(2)
	}
	serviceOrigin := strings.TrimSuffix(serviceURL.String(), "/")
	client := &http.Client{Timeout: deadline.ClientTimeout, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	var req *http.Request
	var err error
	switch *operation {
	case "limits":
		req, err = http.NewRequest("GET", serviceOrigin+"/api/v1/sweeps/limits", nil)
	case "capabilities":
		req, err = http.NewRequest("GET", serviceOrigin+"/api/v1/capabilities", nil)
	case "concepts":
		path := serviceOrigin + "/api/v1/concepts"
		if *query != "" {
			path += "?q=" + url.QueryEscape(*query)
		}
		req, err = http.NewRequest("GET", path, nil)
	case "concept":
		if *conceptID == "" || strings.ContainsAny(*conceptID, "/?#%") {
			fmt.Fprintln(os.Stderr, "valid -concept-id required")
			os.Exit(2)
		}
		req, err = http.NewRequest("GET", serviceOrigin+"/api/v1/concepts/"+*conceptID, nil)
	case "observed-graph", "cpe-preflight", "infer", "sweep-plan", "sweep", "discovery-preview":
		var body []byte
		var r io.Reader = os.Stdin
		if *inputJSON != "-" {
			file, fe := os.Open(*inputJSON)
			if fe != nil {
				fmt.Fprintln(os.Stderr, fe)
				os.Exit(2)
			}
			defer file.Close()
			r = file
		}
		body, err = readOperationInput(r, *operation)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		route := map[string]string{"observed-graph": "/api/v1/query/observed-graph", "cpe-preflight": "/api/v1/cpe/adapter-plan/preflight", "infer": "/api/v1/inference/propose", "sweep-plan": "/api/v1/sweeps/plan", "sweep": "/api/v1/sweeps/execute", "discovery-preview": "/api/v1/sweeps/discovery/preview"}[*operation]
		req, err = http.NewRequest("POST", serviceOrigin+route, bytes.NewReader(body))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
		}
	case "research-evidence":
		path := serviceOrigin + "/api/v1/research-evidence"
		if *researchSection != "" || *query != "" {
			values := url.Values{}
			if *researchSection != "" {
				values.Set("section", *researchSection)
			}
			if *query != "" {
				values.Set("q", *query)
			}
			path += "?" + values.Encode()
		}
		req, err = http.NewRequest("GET", path, nil)
	case "facets":
		if strings.TrimSpace(*query) == "" {
			fmt.Fprintln(os.Stderr, "-query required for facets")
			os.Exit(2)
		}
		if *minSeeders >= 0 || *minLeechers >= 0 || *adult != "any" {
			fmt.Fprintln(os.Stderr, "native facet aggregate cannot honor local-only numeric/adult filters; clear them")
			os.Exit(2)
		}
		groups := []string{}
		for _, group := range strings.Split(*facetGroups, ",") {
			group = strings.TrimSpace(group)
			if group != "" {
				groups = append(groups, group)
			}
		}
		filters := map[string]any{}
		if *languages != "" {
			filters["languages"] = strings.Split(*languages, ",")
		}
		if *media != "" {
			filters["content_types"] = []string{*media}
		}
		payload, _ := json.Marshal(map[string]any{"query": *query, "filters": filters, "groups": groups})
		req, err = http.NewRequest("POST", serviceOrigin+"/api/v1/query/facets", bytes.NewReader(payload))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
		}
	case "plan", "execute":
		if strings.TrimSpace(*query) == "" {
			fmt.Fprintln(os.Stderr, "-query required")
			os.Exit(2)
		}
		terms := []map[string]string{}
		for _, term := range strings.Split(*expansions, ",") {
			term = strings.TrimSpace(term)
			if term != "" {
				terms = append(terms, map[string]string{"term": term, "provenance": "manual-unverified"})
			}
		}
		filters := map[string]any{"adult_metadata": *adult}
		if *languages != "" {
			filters["languages"] = strings.Split(*languages, ",")
		}
		if *media != "" {
			filters["content_types"] = []string{*media}
		}
		if *minSeeders >= 0 {
			filters["min_seeders"] = *minSeeders
		}
		if *minLeechers >= 0 {
			filters["min_leechers"] = *minLeechers
		}
		var conceptIDs []string
		if *selectedConcepts != "" {
			conceptIDs = strings.Split(*selectedConcepts, ",")
		}
		payload, _ := json.Marshal(map[string]any{"query": *query, "expansions": terms, "concept_ids": conceptIDs, "filters": filters, "sort": *sortBy, "group_by": *groupBy, "include_web": *useWeb})
		req, err = http.NewRequest("POST", serviceOrigin+"/api/v1/query/"+*operation, bytes.NewReader(payload))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
		}
	default:
		fmt.Fprintln(os.Stderr, "Invalid operation")
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if authUser != "" {
		req.SetBasicAuth(authUser, authPassword)
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "service unavailable:", err)
		os.Exit(3)
	}
	defer resp.Body.Close()
	output, err := readServiceResponse(resp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	fmt.Println(string(output))
	if resp.StatusCode >= 400 {
		os.Exit(4)
	}
}

func validateServiceOrigin(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("-server must be a valid HTTP(S) origin without credentials, path, query, or fragment")
	}
	if u.Scheme == "http" && !isLoopbackHost(u.Hostname()) {
		return nil, fmt.Errorf("non-loopback service URLs require HTTPS")
	}
	return u, nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func operationInputLimit(operation string) (int64, bool) {
	return contracts.CLIInputLimit(operation)
}

func readOperationInput(r io.Reader, operation string) ([]byte, error) {
	limit, ok := operationInputLimit(operation)
	if !ok {
		return nil, fmt.Errorf("operation %q has no JSON input contract", operation)
	}
	body, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil || len(body) == 0 || int64(len(body)) > limit {
		return nil, fmt.Errorf("bounded JSON input required for %s (maximum %d bytes)", operation, limit)
	}
	return body, nil
}

func readServiceResponse(r io.Reader) ([]byte, error) {
	output, err := io.ReadAll(io.LimitReader(r, maxCLIResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("service response read failed: %w", err)
	}
	if len(output) > maxCLIResponseBytes {
		return nil, fmt.Errorf("service response exceeds %d-byte CLI output budget", maxCLIResponseBytes)
	}
	return output, nil
}
