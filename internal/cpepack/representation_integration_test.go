//go:build !local_stub

package cpepack

import "testing"

// Integration tests exercise the real Build gate, never a test-only source
// projection. A matching hex string is not proof of a trusted CPE release.
func TestBuildSubsetBindsSourcePackageDigest(t *testing.T) {
	p := fixture("fixture.public.walk", "public", nil,
		Concept{ID: "fixture:walk", Definition: "A synthetic gait"})
	lock := lockFor(p)
	selector := SubsetSelector{SchemaVersion: SubsetVersion, ProfileID: "fixture.walk", RootConceptIDs: []string{"fixture:walk"}}
	got, err := BuildSubset([]Package{p}, lock, []string{p.PackageID}, selector, nil, true)
	if err != nil || len(got.Concepts) != 1 || got.Authority != "research_only" {
		t.Fatalf("verified source rejected: view=%+v error=%v", got, err)
	}

	// A caller changing serialized concept content while retaining the pinned
	// source lock cannot obtain a new subset under that old source identity.
	p.Concepts[0].Definition = "forged semantic meaning"
	_, err = BuildSubset([]Package{p}, lock, []string{p.PackageID}, selector, nil, true)
	if err == nil || err.Error() != "content_digest_mismatch" {
		t.Fatalf("forged source admitted: %v", err)
	}
}

func TestBuildSubsetCannotLaunderAcceptedAuthority(t *testing.T) {
	p := fixture("fixture.public.walk", "public", nil,
		Concept{ID: "fixture:walk", Definition: "Research only"})
	selector := SubsetSelector{SchemaVersion: SubsetVersion, ProfileID: "fixture.walk", RootConceptIDs: []string{"fixture:walk"}}
	lock := lockFor(p)
	p.Authority = "accepted"
	_, err := BuildSubset([]Package{p}, lock, []string{p.PackageID}, selector, nil, true)
	if err == nil || err.Error() != "authority_not_admitted" {
		t.Fatalf("accepted CPE authority incorrectly claimed: %v", err)
	}
}
