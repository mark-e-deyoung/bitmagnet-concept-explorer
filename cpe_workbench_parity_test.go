package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark-e-deyoung/bitmagnet-concept-explorer/internal/cpepack"
)

// Exercises EXISTING browser/API/CLI-shared read-model routes, not a second service.
func TestCPEPackageExistingConceptRoutes(t *testing.T) {
	base := filepath.Join("testdata", "cpepack")
	packagesBytes, e := os.ReadFile(filepath.Join(base, "packages.json"))
	if e != nil {
		t.Fatal(e)
	}
	lockBytes, e := os.ReadFile(filepath.Join(base, "lock.json"))
	if e != nil {
		t.Fatal(e)
	}
	var packages []cpepack.Package
	var lock cpepack.Lock
	if e = json.Unmarshal(packagesBytes, &packages); e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(lockBytes, &lock); e != nil {
		t.Fatal(e)
	}
	raw, sha, e := cpepack.BuildBCEPreview(packages, lock, []string{"test.public.literature"})
	if e != nil {
		t.Fatal(e)
	}
	target := filepath.Join(t.TempDir(), "graph.json")
	if e = os.WriteFile(target, raw, 0600); e != nil {
		t.Fatal(e)
	}
	t.Setenv("BCE_CPE_GRAPH_FILE", target)
	t.Setenv("BCE_CPE_GRAPH_SHA256", sha)
	catalog, e := readConceptCatalog()
	if e != nil {
		t.Fatal(e)
	}
	if catalog.AuthorityStatus != "pinned-preview-not-normative" || len(catalog.Concepts) != 2 {
		t.Fatalf("unexpected accepted authority or nodes: %+v", catalog)
	}
	mux := http.NewServeMux()
	registerConceptCatalog(mux)
	for _, test := range []struct {
		path     string
		wantCode int
		want     string
	}{
		{"/api/v1/concepts", 200, "public:literary-work"},
		{"/api/v1/concepts?q=%E0%B8%87%E0%B8%B2%E0%B8%99", 200, "public:work"},
		{"/api/v1/concepts/public:literary-work", 200, "public:literary-work"},
		{"/api/v1/concepts/missing", 404, "concept not in pinned projection"},
	} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, test.path, nil))
		if rec.Code != test.wantCode || !strings.Contains(rec.Body.String(), test.want) {
			t.Fatalf("%s: code=%d body=%s", test.path, rec.Code, rec.Body.String())
		}
	}
	if e = os.WriteFile(target, append(raw, ' '), 0600); e != nil {
		t.Fatal(e)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/concepts", nil))
	if rec.Code != 503 || !strings.Contains(rec.Body.String(), "digest mismatch") {
		t.Fatalf("tampered snapshot unexpectedly served: %d %s", rec.Code, rec.Body.String())
	}
}
