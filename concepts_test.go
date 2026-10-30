package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupFixtureGraph(t *testing.T, raw string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "graph.json")
	if e := os.WriteFile(path, []byte(raw), 0600); e != nil {
		t.Fatal(e)
	}
	hash := sha256.Sum256([]byte(raw))
	t.Setenv("BCE_CPE_GRAPH_FILE", path)
	t.Setenv("BCE_CPE_GRAPH_SHA256", hex.EncodeToString(hash[:]))
}
func TestPinnedReadOnlyConceptGraphAndDrilldown(t *testing.T) {
	setupFixtureGraph(t, `{"schema_version":"bce-graph-preview-v1","snapshot_id":"fixture-001","concepts":[{"id":"media.movie","label":"Movie","definition":"Moving-image work","lexicalizations":[{"language":"de","term":"Film"}],"relations":[{"predicate":"related","target_id":"media.animation"}]},{"id":"media.animation","label":"Animation"}]}`)
	graph, e := readConceptCatalog()
	if e != nil || len(graph.Concepts) != 2 || graph.AuthorityStatus != "pinned-preview-not-normative" {
		t.Fatalf("bad graph %#v %v", graph, e)
	}
	server := httptest.NewServer(handler())
	defer server.Close()
	for path, code := range map[string]int{"/api/v1/concepts": 200, "/api/v1/concepts/media.movie": 200, "/api/v1/concepts/absent": 404} {
		res, e := http.Get(server.URL + path)
		if e != nil {
			t.Fatal(e)
		}
		if res.StatusCode != code {
			t.Errorf("%s status %d want %d", path, res.StatusCode, code)
		}
		res.Body.Close()
	}
}
func TestConceptGraphRejectsMutationOfPinnedFile(t *testing.T) {
	setupFixtureGraph(t, `{"schema_version":"bce-graph-preview-v1","snapshot_id":"fixture","concepts":[]}`)
	path := os.Getenv("BCE_CPE_GRAPH_FILE")
	if e := os.WriteFile(path, []byte(`{"schema_version":"bce-graph-preview-v1","snapshot_id":"changed","concepts":[]}`), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := readConceptCatalog(); e == nil || !strings.Contains(e.Error(), "digest mismatch") {
		t.Fatalf("expected hash mismatch: %v", e)
	}
}
func TestConceptGraphRejectsDanglingReference(t *testing.T) {
	raw := `{"schema_version":"bce-graph-preview-v1","snapshot_id":"fixture","concepts":[{"id":"a","label":"A","relations":[{"predicate":"broader","target_id":"not-present"}]}]}`
	setupFixtureGraph(t, raw)
	if _, e := readConceptCatalog(); e == nil {
		t.Fatal("dangling edge allowed")
	}
}
func TestGraphResponseNoMutationContract(t *testing.T) {
	setupFixtureGraph(t, `{"schema_version":"bce-graph-preview-v1","snapshot_id":"fixture","concepts":[]}`)
	r := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/concepts", nil)
	handler().ServeHTTP(r, req)
	var out ConceptCatalog
	if e := json.Unmarshal(r.Body.Bytes(), &out); e != nil || out.AuthorityStatus != "pinned-preview-not-normative" {
		t.Fatalf("unexpected %#v %v", out, e)
	}
}

func TestSelectedConceptProducesPinnedPreviewTerms(t *testing.T) {
	setupFixtureGraph(t, `{"schema_version":"bce-graph-preview-v1","snapshot_id":"pinned-fixture","concepts":[{"id":"fixture.movie","label":"Movie","lexicalizations":[{"language":"de","term":"Film"},{"language":"th","term":"ภาพยนตร์"}]}]}`)
	p, e := makePlan(Request{Query: "fictional", ConceptIDs: []string{"fixture.movie"}})
	if e != nil || p.ProjectionSnapshot != "pinned-fixture" || len(p.Tracks[1].Terms) != 3 || len(p.ResolvedExpansions) != 3 {
		t.Fatalf("unexpected preview expansion %#v %v", p, e)
	}
	if p.ResolvedExpansions[1].ConceptID != "fixture.movie" || p.ResolvedExpansions[1].Provenance != "digest-pinned-preview:pinned-fixture" {
		t.Fatalf("no lineage %#v", p.ResolvedExpansions)
	}
	if _, e := makePlan(Request{Query: "fictional", ConceptIDs: []string{"invalid"}}); e == nil {
		t.Fatal("accepted unknown concept")
	}
}
func TestMultilingualConceptKeywordQuery(t *testing.T) {
	setupFixtureGraph(t, `{"schema_version":"bce-graph-preview-v1","snapshot_id":"fixture","concepts":[{"id":"fixture.book","label":"Book","lexicalizations":[{"language":"de","term":"Buch"}]},{"id":"fixture.movie","label":"Movie"}]}`)
	recorder := httptest.NewRecorder()
	handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/concepts?q=Buch", nil))
	var graph ConceptCatalog
	if recorder.Code != 200 || json.Unmarshal(recorder.Body.Bytes(), &graph) != nil || len(graph.Concepts) != 1 || graph.Concepts[0].ID != "fixture.book" {
		t.Fatalf("bad search %d: %s", recorder.Code, recorder.Body.String())
	}
}
