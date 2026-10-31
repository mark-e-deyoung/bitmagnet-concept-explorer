package cpepack

import (
	"encoding/json"
	"strings"
	"testing"
)

const commit = "0123456789abcdef0123456789abcdef01234567"

func fixture(id, visibility string, deps []string, cs ...Concept) Package {
	d, e := ContentDigest(cs)
	if e != nil {
		panic(e)
	}
	return Package{PackageID: id, SchemaVersion: PackageVersion, IdentityVersion: IdentityVersion, Visibility: visibility, Authority: "research_only", Repository: "https://github.com/example/public", CommitSHA: commit, Dependencies: deps, ContentSHA256: d, Concepts: cs}
}
func lockFor(ps ...Package) Lock {
	l := Lock{SchemaVersion: LockVersion}
	for _, p := range ps {
		l.Packages = append(l.Packages, LockedPackage{PackageID: p.PackageID, Repository: p.Repository, CommitSHA: p.CommitSHA, ContentSHA256: p.ContentSHA256, SchemaVersion: p.SchemaVersion, IdentityVersion: p.IdentityVersion, Visibility: p.Visibility, Authority: p.Authority, Dependencies: p.Dependencies})
	}
	return l
}
func original() ([]Package, Lock) {
	base := fixture("test.public.base", "public", nil, Concept{ID: "public:work", Definition: "A work", Lexicalizations: []Lexicalization{{"th", "งาน"}, {"en", "work"}}})
	lit := fixture("test.public.literature", "public", []string{base.PackageID}, Concept{ID: "public:literary-work", Definition: "A written work", Relations: []Relation{{"subclass_of", "public:work"}}, Lexicalizations: []Lexicalization{{"de", "Werk"}, {"en", "work"}}})
	private := fixture("test.private.domain", "private", []string{lit.PackageID}, Concept{ID: "private:work", Definition: "An undisclosed extension"})
	return []Package{base, lit, private}, lockFor(base, lit, private)
}
func TestPinnedPublicProjectionDeterministicAndIsolated(t *testing.T) {
	ps, l := original()
	a, err := Build(ps, l, []string{"test.public.literature"}, true)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Build([]Package{ps[2], ps[1], ps[0]}, Lock{SchemaVersion: l.SchemaVersion, Packages: []LockedPackage{l.Packages[2], l.Packages[1], l.Packages[0]}}, []string{"test.public.literature"}, true)
	if err != nil {
		t.Fatal(err)
	}
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	if string(x) != string(y) {
		t.Fatalf("not reproducible: %s vs %s", x, y)
	}
	onlyPublic, err := Build(ps[:2], lockFor(ps[:2]...), []string{"test.public.literature"}, true)
	if err != nil {
		t.Fatal(err)
	}
	z, _ := json.Marshal(onlyPublic)
	if string(x) != string(z) || strings.Contains(string(x), "private") {
		t.Fatal("private unrelated package changed public projection")
	}
	if len(a.Concepts) != 2 || a.Authority != "research_only" {
		t.Fatal("not research-only closure")
	}
}
func TestAdversarialRejection(t *testing.T) {
	cases := []struct {
		name, code string
		mutate     func([]Package, *Lock) []string
	}{
		{"private root", "private_dependency_in_public_closure", func(_ []Package, _ *Lock) []string { return []string{"test.private.domain"} }},
		{"private transitive", "private_dependency_in_public_closure", func(p []Package, l *Lock) []string {
			p[1].Dependencies = []string{p[2].PackageID}
			l.Packages[1].Dependencies = p[1].Dependencies
			p[1].ContentSHA256, _ = ContentDigest(p[1].Concepts)
			return []string{p[1].PackageID}
		}},
		{"missing package", "package_missing", func(p []Package, l *Lock) []string {
			p[1].Dependencies = []string{"test.missing"}
			l.Packages[1].Dependencies = p[1].Dependencies
			return []string{p[1].PackageID}
		}},
		{"cycle", "dependency_cycle", func(p []Package, l *Lock) []string {
			p[0].Dependencies = []string{p[1].PackageID}
			l.Packages[0].Dependencies = p[0].Dependencies
			return []string{p[1].PackageID}
		}},
		{"missing lock", "lock_entry_missing", func(p []Package, l *Lock) []string { l.Packages = l.Packages[:1]; return []string{p[1].PackageID} }},
		{"bad blob digest", "content_digest_mismatch", func(p []Package, _ *Lock) []string {
			p[1].Concepts[0].Definition = "tampered"
			return []string{p[1].PackageID}
		}},
		{"stale origin", "lock_mismatch", func(p []Package, _ *Lock) []string {
			p[1].CommitSHA = "ffffffffffffffffffffffffffffffffffffffff"
			return []string{p[1].PackageID}
		}},
		{"version", "incompatible_schema_or_identity", func(p []Package, _ *Lock) []string {
			p[1].SchemaVersion = "cpe-package-v1"
			return []string{p[1].PackageID}
		}},
		{"acceptance laundering", "authority_not_admitted", func(p []Package, _ *Lock) []string { p[1].Authority = "accepted"; return []string{p[1].PackageID} }},
		{"collision", "concept_identity_collision", func(p []Package, l *Lock) []string {
			p[1].Concepts[0].ID = "public:work"
			p[1].ContentSHA256, _ = ContentDigest(p[1].Concepts)
			l.Packages[1].ContentSHA256 = p[1].ContentSHA256
			return []string{p[1].PackageID}
		}},
		{"unresolved semantic edge", "unresolved_relation", func(p []Package, l *Lock) []string {
			p[1].Concepts[0].Relations[0].TargetID = "public:missing"
			p[1].ContentSHA256, _ = ContentDigest(p[1].Concepts)
			l.Packages[1].ContentSHA256 = p[1].ContentSHA256
			return []string{p[1].PackageID}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, l := original()
			roots := tc.mutate(p, &l)
			_, err := Build(p, l, roots, true)
			if err == nil || err.Error() != tc.code {
				t.Fatalf("want %s got %v", tc.code, err)
			}
		})
	}
}
func TestLexicalHomographNotSemanticIdentity(t *testing.T) {
	p, l := original()
	s, err := Build(p, l, []string{"test.public.literature"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if s.Concepts[0].ID == s.Concepts[1].ID {
		t.Fatal("same translated label must not merge concepts")
	}
}