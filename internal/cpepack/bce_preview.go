package cpepack

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"
)

// BCEPreview is a nonnormative adapter to the existing read-only Explorer contract.
// No public/private data crosses this boundary without Build's transitive closure gate.
type BCEPreview struct {
	SchemaVersion string       `json:"schema_version"`
	SnapshotID    string       `json:"snapshot_id"`
	Concepts      []BCEConcept `json:"concepts"`
}
type BCEConcept struct {
	ID              string           `json:"id"`
	Label           string           `json:"label"`
	Definition      string           `json:"definition,omitempty"`
	Lexicalizations []Lexicalization `json:"lexicalizations,omitempty"`
	Relations       []Relation       `json:"relations,omitempty"`
}

// BuildBCEPreview produces exact deterministic bytes and an immutable content pin
// for BCE_CPE_GRAPH_FILE / BCE_CPE_GRAPH_SHA256. It cannot emit accepted CPE data.
func BuildBCEPreview(packages []Package, lock Lock, roots []string) ([]byte, string, error) {
	projection, err := Build(packages, lock, roots, true)
	if err != nil {
		return nil, "", err
	}
	if projection.Authority != "research_only" || projection.SchemaVersion != SnapshotVersion || !isHex(projection.LockSHA256, 64) {
		return nil, "", errors.New("unqualified_projection")
	}
	if len(projection.Concepts) == 0 || len(projection.Concepts) > 128 {
		return nil, "", errors.New("unsupported_bce_concept_count")
	}
	output := BCEPreview{SchemaVersion: "bce-graph-preview-v1", SnapshotID: "cpe-lock-sha256:" + projection.LockSHA256, Concepts: make([]BCEConcept, 0, len(projection.Concepts))}
	for _, c := range projection.Concepts {
		if len(c.ID) > 128 || len(c.Relations) > 64 || len(c.Lexicalizations) > 64 || !utf8.ValidString(c.Definition) {
			return nil, "", errors.New("unsupported_bce_concept")
		}
		label := ""
		for _, l := range c.Lexicalizations {
			if len(l.Language) > 32 || len([]rune(l.Term)) > 256 || !utf8.ValidString(l.Term) {
				return nil, "", errors.New("unsupported_bce_lexicalization")
			}
			if label == "" {
				label = l.Term
			}
			if strings.EqualFold(l.Language, "en") {
				label = l.Term
				break
			}
		}
		if strings.TrimSpace(label) == "" || len([]rune(label)) > 256 {
			return nil, "", errors.New("missing_bce_display_label")
		}
		for _, r := range c.Relations {
			if len(r.Predicate) > 96 {
				return nil, "", errors.New("unsupported_bce_relation")
			}
		}
		output.Concepts = append(output.Concepts, BCEConcept{ID: c.ID, Label: label, Definition: c.Definition, Lexicalizations: c.Lexicalizations, Relations: c.Relations})
	}
	raw, err := json.Marshal(output)
	if err != nil {
		return nil, "", err
	}
	if len(raw) > 2<<20 {
		return nil, "", errors.New("bce_preview_too_large")
	}
	digest := sha256.Sum256(raw)
	return raw, hex.EncodeToString(digest[:]), nil
}
