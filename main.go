// Concept Explorer: read-only query-plan prototype. No Bitmagnet or CPE writes.
package main

import (
	"crypto/subtle"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/mark-e-deyoung/bitmagnet-concept-explorer/internal/contracts"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

//go:embed web/index.html
var static embed.FS

type Expansion struct {
	Term       string `json:"term"`
	Language   string `json:"language,omitempty"`
	ConceptID  string `json:"concept_id,omitempty"`
	Provenance string `json:"provenance"`
}
type Request struct {
	Query      string       `json:"query"`
	Expansions []Expansion  `json:"expansions,omitempty"`
	ConceptIDs []string     `json:"concept_ids,omitempty"`
	Filters    QueryFilters `json:"filters,omitempty"`
	Sort       string       `json:"sort,omitempty"`
	GroupBy    string       `json:"group_by,omitempty"`
	IncludeWeb bool         `json:"include_web,omitempty"`
}
type Track struct {
	Name  string   `json:"name"`
	Terms []string `json:"terms"`
}
type Plan struct {
	Version            string      `json:"version"`
	ReadOnly           bool        `json:"read_only"`
	Tracks             []Track     `json:"tracks"`
	Warnings           []string    `json:"warnings"`
	ResolvedExpansions []Expansion `json:"resolved_expansions,omitempty"`
	ProjectionSnapshot string      `json:"projection_snapshot,omitempty"`
	ProjectionDigest   string      `json:"projection_digest,omitempty"`
}

func isPinnedProjectionProvenance(value string) bool {
	return strings.HasPrefix(value, "digest-pinned-preview:") || strings.HasPrefix(value, "pinned-preview:")
}

func makePlan(q Request) (Plan, error) {
	if err := validateView(q); err != nil {
		return Plan{}, err
	}
	literal := strings.TrimSpace(q.Query)
	if literal == "" || len([]rune(literal)) > 512 {
		return Plan{}, errors.New("query must contain 1-512 characters")
	}
	direct := Track{Name: "direct", Terms: []string{literal}}
	concept := Track{Name: "concept", Terms: []string{}}
	seen := map[string]bool{strings.ToLower(literal): true}
	resolved := append([]Expansion(nil), q.Expansions...)
	projectionSnapshot, projectionDigest := "", ""
	if len(q.ConceptIDs) > 0 {
		if len(q.ConceptIDs) > 8 {
			return Plan{}, errors.New("too many selected concept IDs")
		}
		graph, err := readConceptCatalog()
		if err != nil {
			return Plan{}, fmt.Errorf("selected concept requires pinned graph: %w", err)
		}
		projectionSnapshot, projectionDigest = graph.SnapshotID, graph.Digest
		for _, id := range q.ConceptIDs {
			var node *ConceptNode
			for i := range graph.Concepts {
				if graph.Concepts[i].ID == id {
					node = &graph.Concepts[i]
					break
				}
			}
			if node == nil {
				return Plan{}, errors.New("selected concept not in pinned projection")
			}
			resolved = append(resolved, Expansion{Term: node.Label, ConceptID: node.ID, Provenance: "digest-pinned-preview:" + graph.SnapshotID})
			for _, lex := range node.Lexicalizations {
				resolved = append(resolved, Expansion{Term: lex.Term, Language: lex.Language, ConceptID: node.ID, Provenance: "digest-pinned-preview:" + graph.SnapshotID})
			}
		}
	}
	if len(resolved) > 128 {
		return Plan{}, errors.New("too many candidate concept expansions")
	}
	used := []Expansion{}
	for _, e := range resolved {
		t := strings.TrimSpace(e.Term)
		if len([]rune(t)) > 512 || t == "" || len(e.Language) > 32 || len(e.ConceptID) > 256 || strings.TrimSpace(e.Provenance) == "" {
			return Plan{}, errors.New("invalid expansion or missing provenance")
		}
		if isPinnedProjectionProvenance(e.Provenance) {
			if err := validateBitmagnetSafeLiteral(e.Term); err != nil {
				return Plan{}, fmt.Errorf("CPE-generated lexicalization is not admitted as a native FTS literal: %w", err)
			}
		}
		if len(concept.Terms) >= 32 {
			return Plan{}, errors.New("too many expansions (maximum 32)")
		}
		key := strings.ToLower(t)
		if !seen[key] {
			concept.Terms = append(concept.Terms, t)
			used = append(used, e)
			seen[key] = true
		}
	}
	return Plan{Version: "v0-provisional", ReadOnly: true, Tracks: []Track{direct, concept}, ResolvedExpansions: used, ProjectionSnapshot: projectionSnapshot, ProjectionDigest: projectionDigest, Warnings: []string{"CPE projection is a local digest-pinned PREVIEW, not normative CPE authority; expansions are retrieval hypotheses only", "Read-only Bitmagnet execution requires a configured target; no CPE intake/promotion is implemented"}}, nil
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func handler() http.Handler {
	mux := http.NewServeMux()
	registerExecution(mux)
	registerSovereign(mux)
	registerNativeFacets(mux)
	registerObservedGraph(mux)
	registerHistoricalCPEPreflight(mux)
	registerHistoricalCPELowerPreview(mux)
	registerSweeps(mux)
	registerSweepDiscovery(mux)
	registerSweepLimits(mux)
	registerProposals(mux)
	registerConceptCatalog(mux)
	registerResearchEvidence(mux)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"status": "ok", "mode": "read-only-optional"})
	})
	mux.HandleFunc("GET /api/v1/capabilities", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"version": "v0-provisional", "query_plan": true, "execute": bitmagnetEndpointConfigured(), "execution_target_validation": "configuration_only_no_live_probe", "discovery_preview": true, "metadata_facets": true, "native_facet_aggregations": true, "observed_result_lexical_graph": true, "historical_cpe_adapter_plan_preflight": true, "historical_cpe_fts_lower_preview": true, "multilingual_sweep_plan": true, "multilingual_sweep_execute": true, "multilingual_sweep_limits": true, "searxng_sweep_requires_private_operator_consent": true, "cpe_discovery_research_preview_only": true, "sort": true, "grouping": true, "concept_graph_configured": os.Getenv("BCE_CPE_GRAPH_FILE") != "" && os.Getenv("BCE_CPE_GRAPH_SHA256") != "", "searxng_configured": os.Getenv("BCE_SEARXNG_URL") != "", "public_research_evidence_read_only": true, "public_research_evidence_configured": os.Getenv("BCE_RESEARCH_EVIDENCE_FILE") != "" && os.Getenv("BCE_RESEARCH_EVIDENCE_SHA256") != "", "cpe_write": false, "mutation": false, "sovereign_inference_configured": os.Getenv("BCE_SOVEREIGN_BASE_URL") != "" && os.Getenv("BCE_SOVEREIGN_MODEL") != "", "sovereign_inference_authority": "hypotheses_only_no_policy_or_cpe_write", "audiences": []string{"human", "automation", "agent"}})
	})
	mux.HandleFunc("POST /api/v1/query/plan", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, contracts.QueryJSONBytes)
		defer r.Body.Close()
		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields()
		var q Request
		if err := d.Decode(&q); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid JSON or body too large"})
			return
		}
		if err := d.Decode(new(any)); err != io.EOF {
			writeJSON(w, 400, map[string]string{"error": "unexpected trailing JSON"})
			return
		}
		p, err := makePlan(q)
		if err != nil {
			writeJSON(w, 422, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, p)
	})
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data, err := static.ReadFile("web/index.html")
		if err != nil {
			http.Error(w, "unavailable", 500)
			return
		}
		_, _ = w.Write(data)
	})
	return mux
}

func validateListenAuth(addr, user, password string) error {
	if (user == "") != (password == "") {
		return errors.New("BCE_AUTH_USER and BCE_AUTH_PASSWORD must be configured together")
	}
	if user != "" {
		if len(user) > 256 || strings.ContainsAny(user, ":\r\n") || len(password) < 16 || len(password) > 1024 || strings.ContainsAny(password, "\r\n") {
			return errors.New("invalid Basic Auth credentials: username must be bounded and password must contain 16-1024 bytes")
		}
	}
	if !isLoopbackBind(addr) && user == "" {
		return errors.New("non-loopback bind requires BCE_AUTH_USER and BCE_AUTH_PASSWORD")
	}
	return nil
}

func authenticated(next http.Handler, user, password string) http.Handler {
	if user == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" && r.Method == http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}
		gotUser, gotPassword, ok := r.BasicAuth()
		if !ok || subtle.ConstantTimeCompare([]byte(gotUser), []byte(user)) != 1 || subtle.ConstantTimeCompare([]byte(gotPassword), []byte(password)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="Concept Explorer", charset="UTF-8"`)
			w.Header().Set("Cache-Control", "no-store")
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	addr := flag.String("listen", "127.0.0.1:3086", "bind address, default loopback")
	healthcheck := flag.Bool("healthcheck", false, "check the local health endpoint and exit")
	flag.Parse()
	if os.Getenv("BCE_LISTEN") != "" {
		*addr = os.Getenv("BCE_LISTEN")
	}
	if *healthcheck {
		if err := checkHealth(*addr); err != nil {
			log.Fatal(err)
		}
		return
	}
	user, password := os.Getenv("BCE_AUTH_USER"), os.Getenv("BCE_AUTH_PASSWORD")
	if err := validateListenAuth(*addr, user, password); err != nil {
		log.Fatal(err)
	}
	server := &http.Server{Addr: *addr, Handler: authenticated(handler(), user, password), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: sweepHTTPWriteTimeout, IdleTimeout: 30 * time.Second}
	log.Printf("concept explorer prototype listening on %s", *addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(fmt.Errorf("serve: %w", err))
	}
}

func checkHealth(addr string) error {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return errors.New("invalid healthcheck listen address")
	}
	u := url.URL{Scheme: "http", Host: net.JoinHostPort("127.0.0.1", port), Path: "/healthz"}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(u.String())
	if err != nil {
		return errors.New("health endpoint unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health endpoint returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// Report only an admitted configured endpoint, not reachability or live readiness.
func bitmagnetEndpointConfigured() bool {
	_, err := endpoint()
	return err == nil
}

func isLoopbackBind(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
