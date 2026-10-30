package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestObservedGraphOnlyEvidenceBackedNodes(t *testing.T) {
	catalog := ConceptCatalog{Digest: "sha256:fixture", AuthorityStatus: "pinned-preview-not-normative", Concepts: []ConceptNode{
		{ID: "genre.cyberpunk", Label: "Cyberpunk", Relations: []ConceptRelation{{Predicate: "related", TargetID: "genre.dystopian"}}},
		{ID: "genre.dystopian", Label: "Dystopian"},
		{ID: "media.game", Label: "Game"},
		{ID: "format.rom", Label: "ROM"},
	}}
	records := []Observation{{InfoHash: "hash1", Title: "Cyberpunk Game demo"}, {InfoHash: "hash1", Title: "Cyberpunk Game demo"}, {InfoHash: "hash2", Title: "Cyberpunk Fromage"}}
	g, e := buildObservedGraph(catalog, records)
	if e != nil {
		t.Fatal(e)
	}
	if g.Status != "lexical_candidates_observed" || len(g.Concepts) != 2 {
		t.Fatalf("ghost related concepts or no evidence: %#v", g)
	}
	if len(g.EvidenceEdges) != 3 {
		t.Fatalf("duplicate resource-concept trace was counted: %#v", g.EvidenceEdges)
	}
	counts := map[string]int{}
	for _, v := range g.Concepts {
		counts[v.ID] = v.DistinctReturnedTorrents
	}
	if counts["genre.cyberpunk"] != 2 || counts["media.game"] != 1 || counts["genre.dystopian"] != 0 || counts["format.rom"] != 0 {
		t.Fatalf("wrong unique counts: %#v", counts)
	}
	for _, edge := range g.EvidenceEdges {
		if edge.AssessmentEligible || edge.EvidenceRole != "candidate_lexical_match" {
			t.Fatal("lexical title promoted to semantic truth")
		}
	}
}
func TestObservedGraphUnicodeAndBoundaries(t *testing.T) {
	tests := []struct {
		title, term string
		want        bool
	}{
		{"Fromage", "ROM", false},
		{"ROM files", "ROM", true},
		{"cyberpunk", "Cyberpunk", true},
		{"Cyberpunking", "Cyberpunk", false},
		{"Das Buch", "buch", true},
		{"映画 アニメーション", "アニメーション", true},
		{"ภาษาไทย ภาพยนตร์", "ภาพยนตร์", true},
	}
	for _, v := range tests {
		if got := boundedLexicalHit(v.title, v.term); got != v.want {
			t.Errorf("title=%q term=%q got=%v want=%v", v.title, v.term, got, v.want)
		}
	}
}
func TestObservedGraphNoMatchesIsNotSemanticAbsence(t *testing.T) {
	c := ConceptCatalog{Digest: "sha256:fixture", Concepts: []ConceptNode{{ID: "genre.scifi", Label: "Science fiction"}}}
	g, e := buildObservedGraph(c, []Observation{{InfoHash: "x", Title: "unknown metadata"}})
	if e != nil || g.Status != "no_lexical_evidence" || len(g.Concepts) != 0 || g.ReturnedRecords != 1 {
		t.Fatalf("unsafe no-match interpretation %#v %v", g, e)
	}
	g, e = buildObservedGraph(c, nil)
	if e != nil || g.Status != "no_returned_records" {
		t.Fatalf("confused empty return %#v %v", g, e)
	}
}
func TestObservedGraphAPIConsumesExistingReceiptWithoutRefetch(t *testing.T) {
	setupFixtureGraph(t, `{"schema_version":"bce-graph-preview-v1","snapshot_id":"fixture","concepts":[{"id":"media.game","label":"Game"}]}`)
	r := httptest.NewRecorder()
	handler().ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/v1/query/observed-graph", bytes.NewBufferString(`{"results":[{"info_hash":"fake-01","title":"Game example"}]}`)))
	if r.Code != 200 {
		t.Fatalf("API status %d: %s", r.Code, r.Body.String())
	}
	var out ObservedGraph
	if e := json.Unmarshal(r.Body.Bytes(), &out); e != nil || len(out.Concepts) != 1 || out.Concepts[0].ID != "media.game" {
		t.Fatalf("bad observed API: %s / %v", r.Body.String(), e)
	}
	if out.Scope != "supplied_bounded_bitmagnet_result_metadata_not_corpus_global" {
		t.Fatalf("scope misrepresented: %#v", out)
	}
}
