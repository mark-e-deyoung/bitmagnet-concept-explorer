package main

import (
    "bytes"
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

func pinResearchFixture(t *testing.T, content []byte) {
    t.Helper()
    path := filepath.Join(t.TempDir(), "research-evidence.json")
    if err := os.WriteFile(path, content, 0600); err != nil { t.Fatal(err) }
    sum := sha256.Sum256(content)
    t.Setenv("BCE_RESEARCH_EVIDENCE_FILE", path)
    t.Setenv("BCE_RESEARCH_EVIDENCE_SHA256", hex.EncodeToString(sum[:]))
}

func TestResearchEvidencePinnedResearchOnly(t *testing.T) {
    raw, err := os.ReadFile(filepath.Join(publicPackDir, "catalog-evidence.json"))
    if err != nil { t.Fatal(err) }
    pinResearchFixture(t, raw)
    p, digest, err := readResearchEvidence()
    if err != nil { t.Fatal(err) }
    if p.AuthorityStatus != "research-only-no-accepted-cpe-facts" ||
       len(p.Works) != 4 || len(p.Entities) != 4 || len(p.Claims) != 5 ||
       len(p.ArtCandidates) != 3 || len(p.SourceInventory) != 7 ||
       !strings.HasPrefix(digest,"sha256:") {
        t.Fatalf("unexpected source-research data: %+v %q", p, digest)
    }
    for _, art := range p.ArtCandidates {
        if art.Downloaded || art.FileRetained { t.Fatal("media must not be retained") }
    }
    req := httptest.NewRequest(http.MethodGet, "/api/v1/research-evidence?section=art_candidates&q=Schoonover", nil)
    w := httptest.NewRecorder()
    handler().ServeHTTP(w, req)
    if w.Code != 200 { t.Fatalf("research API failed: %d %s", w.Code, w.Body.String()) }
    var response map[string]any
    if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil { t.Fatal(err) }
    if response["authority_status"] != p.AuthorityStatus || response["count"] != float64(1) {
        t.Fatalf("wrong bounded source-only evidence response: %+v", response)
    }
    req = httptest.NewRequest(http.MethodGet, "/api/v1/research-evidence?section=claims&q=barsoom", nil)
    w = httptest.NewRecorder()
    handler().ServeHTTP(w, req)
    if w.Code != 200 { t.Fatal(w.Code, w.Body.String()) }
    req = httptest.NewRequest(http.MethodGet, "/api/v1/research-evidence?section=not-a-section", nil)
    w = httptest.NewRecorder()
    handler().ServeHTTP(w, req)
    if w.Code != 422 { t.Fatalf("accepted invalid section: %d", w.Code) }
}

func TestResearchEvidenceRejectsWrongDigestAndPromotion(t *testing.T) {
    raw, err := os.ReadFile(filepath.Join(publicPackDir, "catalog-evidence.json"))
    if err != nil { t.Fatal(err) }
    pinResearchFixture(t, raw)
    t.Setenv("BCE_RESEARCH_EVIDENCE_SHA256", strings.Repeat("0", 64))
    if _, _, err := readResearchEvidence(); err == nil {
        t.Fatal("wrong hash was accepted")
    }
    promoted := bytes.Replace(raw, []byte("research-only-no-accepted-cpe-facts"), []byte("accepted"), 1)
    pinResearchFixture(t, promoted)
    if _, _, err := readResearchEvidence(); err == nil {
        t.Fatal("unqualified fact promotion was accepted")
    }
    retained := bytes.Replace(raw, []byte("\"file_retained\": false"), []byte("\"file_retained\": true"), 1)
    pinResearchFixture(t, retained)
    if _, _, err := readResearchEvidence(); err == nil {
        t.Fatal("retained media was accepted")
    }
    orphan := bytes.Replace(raw, []byte("\"subject\": \"work:princess-of-mars\""), []byte("\"subject\": \"work:orphan\""), 1)
    pinResearchFixture(t, orphan)
    if _, _, err := readResearchEvidence(); err == nil {
        t.Fatal("orphaned claim was accepted")
    }
}

func TestResearchEvidenceNoDefaultAndNoWrites(t *testing.T) {
    t.Setenv("BCE_RESEARCH_EVIDENCE_FILE", "")
    t.Setenv("BCE_RESEARCH_EVIDENCE_SHA256", "")
    req := httptest.NewRequest(http.MethodGet, "/api/v1/research-evidence", nil)
    w := httptest.NewRecorder()
    handler().ServeHTTP(w, req)
    if w.Code != 503 { t.Fatalf("unexpected unconfigured source response: %d", w.Code) }
    req = httptest.NewRequest(http.MethodPost, "/api/v1/research-evidence", bytes.NewReader([]byte("{}")))
    w = httptest.NewRecorder()
    handler().ServeHTTP(w, req)
    if w.Code != 405 { t.Fatalf("research endpoint unexpectedly permits writes: %d", w.Code) }
}
