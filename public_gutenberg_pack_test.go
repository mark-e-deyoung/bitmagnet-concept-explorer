package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const publicPackDir = "testdata/public-cpe-packs/gutenberg-lore-v0"

func TestPublicGutenbergGraphDigestResearchOnly(t *testing.T) {
	path := filepath.Join(publicPackDir, "graph-preview.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	t.Setenv("BCE_CPE_GRAPH_FILE", path)
	t.Setenv("BCE_CPE_GRAPH_SHA256", hex.EncodeToString(sum[:]))
	graph, err := readConceptCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if graph.AuthorityStatus != "pinned-preview-not-normative" || len(graph.Concepts) < 5 {
		t.Fatalf("wrong graph authority/size: %+v", graph)
	}
	t.Setenv("BCE_CPE_GRAPH_SHA256", strings.Repeat("0", 64))
	if _, err = readConceptCatalog(); err == nil {
		t.Fatal("mismatched graph digest accepted")
	}
}

func TestPublicGutenbergEvidenceSafeAndConnected(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(publicPackDir, "catalog-evidence.json"))
	if err != nil {
		t.Fatal(err)
	}
	var p map[string]any
	if err = json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	if p["authority_status"] != "research-only-no-accepted-cpe-facts" {
		t.Fatal("factual authority incorrectly promoted")
	}
	works, ok := p["works"].([]any)
	if !ok || len(works) != 4 {
		t.Fatal("expected four public-domain candidate works")
	}
	known := map[string]bool{}
	for _, v := range works {
		w := v.(map[string]any)
		id := w["id"].(string)
		if id == "" || known[id] || !strings.HasPrefix(w["source_url"].(string), "https://www.gutenberg.org/ebooks/") {
			t.Fatal("invalid work identity/source")
		}
		known[id] = true
	}
	claims := p["claims"].([]any)
	seen := map[string]bool{}
	for _, v := range claims {
		c := v.(map[string]any)
		id := c["id"].(string)
		if id == "" || seen[id] || !known[c["subject"].(string)] || !strings.HasPrefix(c["source_url"].(string), "https://") {
			t.Fatal("unattributed or orphaned claim")
		}
		seen[id] = true
	}
	arts := p["art_candidates"].([]any)
	if len(arts) < 2 {
		t.Fatal("no specific art candidates")
	}
	for _, v := range arts {
		a := v.(map[string]any)
		if !known[a["related_work"].(string)] || !strings.HasPrefix(a["source_url"].(string), "https://") || a["downloaded"] != false || a["file_retained"] != false {
			t.Fatal("orphan, retained or unsupported art asset")
		}
	}
}
