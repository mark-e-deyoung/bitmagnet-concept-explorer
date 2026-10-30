package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// ConceptSnapshot is a PROVISIONAL *read-only* display interchange. It is not
// the normative CPE schema or proof that any concept has been accepted.
type ConceptSnapshot struct {
	SchemaVersion string        `json:"schema_version"`
	SnapshotID    string        `json:"snapshot_id"`
	Concepts      []ConceptNode `json:"concepts"`
}
type ConceptNode struct {
	ID              string            `json:"id"`
	Label           string            `json:"label"`
	Definition      string            `json:"definition,omitempty"`
	Lexicalizations []Lexeme          `json:"lexicalizations,omitempty"`
	Relations       []ConceptRelation `json:"relations,omitempty"`
}
type Lexeme struct {
	Language string `json:"language"`
	Term     string `json:"term"`
}
type ConceptRelation struct {
	Predicate string `json:"predicate"`
	TargetID  string `json:"target_id"`
}
type ConceptCatalog struct {
	SchemaVersion   string        `json:"schema_version"`
	SnapshotID      string        `json:"snapshot_id"`
	Digest          string        `json:"digest"`
	AuthorityStatus string        `json:"authority_status"`
	Concepts        []ConceptNode `json:"concepts"`
	Warnings        []string      `json:"warnings"`
}

func readConceptCatalog() (ConceptCatalog, error) {
	path := strings.TrimSpace(os.Getenv("BCE_CPE_GRAPH_FILE"))
	expected := strings.ToLower(strings.TrimSpace(os.Getenv("BCE_CPE_GRAPH_SHA256")))
	if path == "" || len(expected) != 64 {
		return ConceptCatalog{}, errors.New("pinned CPE graph projection not configured")
	}
	if _, e := hex.DecodeString(expected); e != nil {
		return ConceptCatalog{}, errors.New("invalid expected CPE graph digest")
	}
	f, e := os.Open(path)
	if e != nil {
		return ConceptCatalog{}, errors.New("CPE graph projection unavailable")
	}
	defer f.Close()
	raw, e := io.ReadAll(io.LimitReader(f, 2<<20+1))
	if e != nil || len(raw) > 2<<20 {
		return ConceptCatalog{}, errors.New("CPE graph projection too large or unreadable")
	}
	hash := sha256.Sum256(raw)
	actual := hex.EncodeToString(hash[:])
	if actual != expected {
		return ConceptCatalog{}, errors.New("CPE graph projection digest mismatch")
	}
	var graph ConceptSnapshot
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if e := dec.Decode(&graph); e != nil {
		return ConceptCatalog{}, errors.New("invalid CPE graph projection")
	}
	var trailing any
	if e := dec.Decode(&trailing); e != io.EOF {
		return ConceptCatalog{}, errors.New("trailing CPE graph data")
	}
	if graph.SchemaVersion != "bce-graph-preview-v1" || graph.SnapshotID == "" || len(graph.SnapshotID) > 256 || len(graph.Concepts) > 128 {
		return ConceptCatalog{}, errors.New("unsupported provisional graph projection")
	}
	seen := map[string]bool{}
	for _, n := range graph.Concepts {
		if n.ID == "" || len(n.ID) > 128 || n.Label == "" || len([]rune(n.Label)) > 256 || seen[n.ID] || len(n.Relations) > 64 || len(n.Lexicalizations) > 64 {
			return ConceptCatalog{}, errors.New("invalid or duplicated concept node")
		}
		seen[n.ID] = true
		for _, l := range n.Lexicalizations {
			if l.Language == "" || len(l.Language) > 32 || l.Term == "" || len([]rune(l.Term)) > 256 {
				return ConceptCatalog{}, errors.New("invalid lexicalization")
			}
		}
	}
	for _, n := range graph.Concepts {
		for _, edge := range n.Relations {
			if edge.Predicate == "" || len(edge.Predicate) > 96 || !seen[edge.TargetID] {
				return ConceptCatalog{}, fmt.Errorf("invalid relationship target or predicate for %q", n.ID)
			}
		}
	}
	return ConceptCatalog{SchemaVersion: graph.SchemaVersion, SnapshotID: graph.SnapshotID, Digest: "sha256:" + actual,
		AuthorityStatus: "pinned-preview-not-normative", Concepts: graph.Concepts,
		Warnings: []string{"A matching digest pins the imported file, but does not certify CPE semantic acceptance or grant write authority"}}, nil
}
func registerConceptCatalog(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/concepts", func(w http.ResponseWriter, r *http.Request) {
		graph, e := readConceptCatalog()
		if e != nil {
			writeJSON(w, 503, map[string]string{"error": e.Error()})
			return
		}
		q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
		if len([]rune(q)) > 256 {
			writeJSON(w, 422, map[string]string{"error": "concept search term too long"})
			return
		}
		if q != "" {
			matched := []ConceptNode{}
			for _, node := range graph.Concepts {
				found := strings.Contains(strings.ToLower(node.ID), q) || strings.Contains(strings.ToLower(node.Label), q) || strings.Contains(strings.ToLower(node.Definition), q)
				for _, lex := range node.Lexicalizations {
					if strings.Contains(strings.ToLower(lex.Term), q) {
						found = true
						break
					}
				}
				if found {
					matched = append(matched, node)
				}
			}
			graph.Concepts = matched
		}
		writeJSON(w, 200, graph)
	})
	mux.HandleFunc("GET /api/v1/concepts/{id}", func(w http.ResponseWriter, r *http.Request) {
		graph, e := readConceptCatalog()
		if e != nil {
			writeJSON(w, 503, map[string]string{"error": e.Error()})
			return
		}
		id := r.PathValue("id")
		for _, n := range graph.Concepts {
			if n.ID == id {
				writeJSON(w, 200, map[string]any{
					"snapshot_id": graph.SnapshotID, "digest": graph.Digest, "authority_status": graph.AuthorityStatus, "concept": n,
				})
				return
			}
		}
		writeJSON(w, 404, map[string]string{"error": "concept not in pinned projection"})
	})
}
