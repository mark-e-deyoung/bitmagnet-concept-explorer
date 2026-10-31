// Package cpepack qualifies research-only Git packages and immutable locks.
// It does not accept concepts, adjudicate facts, or publish artifacts.
package cpepack

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	PackageVersion  = "cpe-package-v0"
	IdentityVersion = "cpe-package-id-v0"
	LockVersion     = "cpe-lock-v0"
	SnapshotVersion = "cpe-readmodel-v0"
)

type Lexicalization struct {
	Language string `json:"language"`
	Term     string `json:"term"`
}
type Relation struct {
	Predicate string `json:"predicate"`
	TargetID  string `json:"target_id"`
}
type Concept struct {
	ID              string           `json:"id"`
	Definition      string           `json:"definition"`
	Lexicalizations []Lexicalization `json:"lexicalizations,omitempty"`
	Relations       []Relation       `json:"relations,omitempty"`
}
type Package struct {
	PackageID       string    `json:"package_id"`
	SchemaVersion   string    `json:"schema_version"`
	IdentityVersion string    `json:"identity_version"`
	Visibility      string    `json:"visibility"`
	Authority       string    `json:"authority"`
	Repository      string    `json:"repository"`
	CommitSHA       string    `json:"commit_sha"`
	Dependencies    []string  `json:"dependencies,omitempty"`
	ContentSHA256   string    `json:"content_sha256"`
	Concepts        []Concept `json:"concepts"`
}
type LockedPackage struct {
	PackageID       string   `json:"package_id"`
	Repository      string   `json:"repository"`
	CommitSHA       string   `json:"commit_sha"`
	ContentSHA256   string   `json:"content_sha256"`
	SchemaVersion   string   `json:"schema_version"`
	IdentityVersion string   `json:"identity_version"`
	Visibility      string   `json:"visibility"`
	Authority       string   `json:"authority"`
	Dependencies    []string `json:"dependencies,omitempty"`
}
type Lock struct {
	SchemaVersion string          `json:"schema_version"`
	Packages      []LockedPackage `json:"packages"`
}
type Projection struct {
	SchemaVersion string    `json:"schema_version"`
	Authority     string    `json:"authority"`
	LockSHA256    string    `json:"lock_sha256"`
	Concepts      []Concept `json:"concepts"`
}

func fail(code string) error { return errors.New(code) }
func sha(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func isHex(s string, n int) bool {
	if len(s) != n {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func validID(s string) bool {
	if len(s) == 0 || len(s) > 160 {
		return false
	}
	for i, c := range s {
		if c > 127 || !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || (i > 0 && (c == ':' || c == '.' || c == '-' || c == '_'))) {
			return false
		}
	}
	return true
}
// v0 accepts a narrow GitHub HTTPS origin grammar, without credentials, ports,
// escapes or query fragments. Other Git forges require a versioned extension.
func validGitOrigin(source string) bool {
	u, err := url.Parse(source)
	if err != nil || u.Scheme != "https" || u.Hostname() != "github.com" || u.Host != "github.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || u.RawPath != "" {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) != 2 {
		return false
	}
	for _, p := range parts {
		if len(p) == 0 || len(p) > 100 || p == "." || p == ".." || strings.Contains(p, "..") {
			return false
		}
		for _, r := range p {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.') {
				return false
			}
		}
	}
	return true
}
func sortedUnique(items []string) ([]string, error) {
	out := append([]string{}, items...)
	sort.Strings(out)
	for i, v := range out {
		if !validID(v) || (i > 0 && out[i-1] == v) {
			return nil, fail("invalid_or_duplicate_dependency")
		}
	}
	return out, nil
}
func normalizeConcepts(input []Concept) ([]Concept, error) {
	out := append([]Concept{}, input...)
	for i := range out {
		c := &out[i]
		if !validID(c.ID) || strings.TrimSpace(c.Definition) == "" || !utf8.ValidString(c.Definition) {
			return nil, fail("invalid_concept")
		}
		c.Lexicalizations = append([]Lexicalization{}, c.Lexicalizations...)
		sort.Slice(c.Lexicalizations, func(i, j int) bool {
			a, b := c.Lexicalizations[i], c.Lexicalizations[j]
			if a.Language != b.Language {
				return a.Language < b.Language
			}
			return a.Term < b.Term
		})
		for j, l := range c.Lexicalizations {
			if !utf8.ValidString(l.Language) || !utf8.ValidString(l.Term) || strings.TrimSpace(l.Language) == "" || strings.TrimSpace(l.Term) == "" || (j > 0 && l == c.Lexicalizations[j-1]) {
				return nil, fail("invalid_lexicalization")
			}
		}
		c.Relations = append([]Relation{}, c.Relations...)
		sort.Slice(c.Relations, func(i, j int) bool {
			a, b := c.Relations[i], c.Relations[j]
			if a.Predicate != b.Predicate {
				return a.Predicate < b.Predicate
			}
			return a.TargetID < b.TargetID
		})
		for j, r := range c.Relations {
			if !validID(r.Predicate) || !validID(r.TargetID) || (j > 0 && r == c.Relations[j-1]) {
				return nil, fail("invalid_relation")
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	for i := 1; i < len(out); i++ {
		if out[i-1].ID == out[i].ID {
			return nil, fail("duplicate_concept")
		}
	}
	return out, nil
}

// ContentDigest is byte-stable across semantically unordered concept, lexical and edge arrays.
// This v0 struct encoding is intentionally NOT the pre-existing sweep-id-v1 algorithm.
func ContentDigest(concepts []Concept) (string, error) {
	normalized, err := normalizeConcepts(concepts)
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return sha(raw), nil
}
func same(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
func selectedLockDigest(entries []LockedPackage) (string, error) {
	sort.Slice(entries, func(i, j int) bool { return entries[i].PackageID < entries[j].PackageID })
	raw, err := json.Marshal(Lock{SchemaVersion: LockVersion, Packages: entries})
	if err != nil {
		return "", err
	}
	return sha(raw), nil
}

// Build refuses missing, stale, cyclic, incompatible and privacy-violating dependencies.
// Only the requested transitive closure is included in the public digest/output.
func Build(packages []Package, lock Lock, roots []string, public bool) (Projection, error) {
	empty := Projection{}
	if lock.SchemaVersion != LockVersion || len(roots) == 0 {
		return empty, fail("invalid_lock")
	}
	byID := map[string]Package{}
	for _, p := range packages {
		if !validID(p.PackageID) || byID[p.PackageID].PackageID != "" {
			return empty, fail("duplicate_package")
		}
		byID[p.PackageID] = p
	}
	pinned := map[string]LockedPackage{}
	for _, p := range lock.Packages {
		if !validID(p.PackageID) || pinned[p.PackageID].PackageID != "" {
			return empty, fail("duplicate_lock_entry")
		}
		pinned[p.PackageID] = p
	}
	state := map[string]uint8{}
	used := map[string]LockedPackage{}
	concepts := map[string]Concept{}
	var visit func(string) error
	visit = func(id string) error {
		if state[id] == 1 {
			return fail("dependency_cycle")
		}
		if state[id] == 2 {
			return nil
		}
		p, ok := byID[id]
		if !ok {
			return fail("package_missing")
		}
		l, ok := pinned[id]
		if !ok {
			return fail("lock_entry_missing")
		}
		if p.PackageID != l.PackageID || p.SchemaVersion != PackageVersion || p.IdentityVersion != IdentityVersion || l.SchemaVersion != PackageVersion || l.IdentityVersion != IdentityVersion {
			return fail("incompatible_schema_or_identity")
		}
		if p.Authority != "research_only" || l.Authority != "research_only" {
			return fail("authority_not_admitted")
		}
		if p.Visibility != "public" && p.Visibility != "private" {
			return fail("invalid_visibility")
		}
		if public && (p.Visibility != "public" || l.Visibility != "public") {
			return fail("private_dependency_in_public_closure")
		}
		if !validGitOrigin(p.Repository) {
			return fail("invalid_git_origin")
		}
		if p.Visibility != l.Visibility || p.Repository != l.Repository || p.CommitSHA != l.CommitSHA || p.ContentSHA256 != l.ContentSHA256 || !isHex(p.CommitSHA, 40) || !isHex(p.ContentSHA256, 64) {
			return fail("lock_mismatch")
		}
		pd, err := sortedUnique(p.Dependencies)
		if err != nil {
			return err
		}
		ld, err := sortedUnique(l.Dependencies)
		if err != nil {
			return err
		}
		if !same(pd, ld) {
			return fail("dependency_lock_mismatch")
		}
		digest, err := ContentDigest(p.Concepts)
		if err != nil {
			return err
		}
		if digest != p.ContentSHA256 {
			return fail("content_digest_mismatch")
		}
		state[id] = 1
		for _, dep := range pd {
			if err := visit(dep); err != nil {
				return err
			}
		}
		normalized, _ := normalizeConcepts(p.Concepts)
		for _, c := range normalized {
			if _, exists := concepts[c.ID]; exists {
				return fail("concept_identity_collision")
			}
			concepts[c.ID] = c
		}
		l.Dependencies = ld
		used[id] = l
		state[id] = 2
		return nil
	}
	for _, r := range roots {
		if err := visit(r); err != nil {
			return empty, err
		}
	}
	sorted := make([]Concept, 0, len(concepts))
	for _, c := range concepts {
		sorted = append(sorted, c)
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	for _, c := range sorted {
		for _, r := range c.Relations {
			if _, ok := concepts[r.TargetID]; !ok {
				return empty, fail("unresolved_relation")
			}
		}
	}
	entries := make([]LockedPackage, 0, len(used))
	for _, l := range used {
		entries = append(entries, l)
	}
	digest, err := selectedLockDigest(entries)
	if err != nil {
		return empty, err
	}
	return Projection{SchemaVersion: SnapshotVersion, Authority: "research_only", LockSHA256: digest, Concepts: sorted}, nil
}
