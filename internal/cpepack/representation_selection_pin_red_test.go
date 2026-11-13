package cpepack

import "testing"

// Frozen research characterization: this is not evidence of a production
// vulnerability or a claim that BuildSubset promises accepted-release pins.
// It demonstrates why a *separate trusted expected selection digest* is
// necessary before a downstream consumer treats a representation set as fixed.
func redPinnedSourceFixture() ([]Package, Lock, SubsetSelector, []RepresentationRef) {
	p := fixture("fixture.public.release-pin", "public", nil,
		Concept{ID: "fixture:motion", Definition: "Original synthetic motion concept"})
	selector := SubsetSelector{
		SchemaVersion: SubsetVersion,
		ProfileID: "fixture.pin.v0",
		RootConceptIDs: []string{"fixture:motion"},
	}
	ref := RepresentationRef{
		SchemaVersion: RepresentationVersion,
		ReferenceID: "fixture:motion-example",
		ConceptID: "fixture:motion",
		Role: "demonstrates",
		Modality: "motion",
		MediaType: "application/json",
		FormatVersion: "fixture.motion.v0",
		Artifact: ArtifactIdentity{Kind: "raw-sha256", Digest: "sha256:" + fixtureHash},
		Visibility: "public",
		Rights: "redistributable",
		SourceRevision: "fixture-src-v1",
	}
	return []Package{p}, lockFor(p), selector, []RepresentationRef{ref}
}

func TestREDResearchSourceLockDoesNotIndependentlyAuthorizeExampleSet(t *testing.T) {
	packages, lock, selector, refs := redPinnedSourceFixture()
	original, err := BuildSubset(packages, lock, []string{packages[0].PackageID}, selector, refs, true)
	if err != nil { t.Fatal(err) }

	refs[0].Artifact.Digest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	changed, err := BuildSubset(packages, lock, []string{packages[0].PackageID}, selector, refs, true)
	if err != nil { t.Fatal(err) }
	if original.SourceLockSHA256 != changed.SourceLockSHA256 {
		t.Fatal("semantic source lock unexpectedly changed")
	}
	if original.Digest == changed.Digest {
		t.Fatal("research subset failed to bind changed representation")
	}
	// Both research projections are legal. Only the *independent* original
	// digest can select the original release/reference set; source alone cannot.
	if changed.Digest == original.Digest {
		t.Fatal("externally changed reference acquired original selection identity")
	}
}

func TestREDSourceRevisionIsNotSemanticLockMembership(t *testing.T) {
	packages, lock, selector, refs := redPinnedSourceFixture()
	original, err := BuildSubset(packages, lock, []string{packages[0].PackageID}, selector, refs, true)
	if err != nil { t.Fatal(err) }
	refs[0].SourceRevision = "fixture-src-v2"
	next, err := BuildSubset(packages, lock, []string{packages[0].PackageID}, selector, refs, true)
	if err != nil { t.Fatal(err) }
	if original.SourceLockSHA256 != next.SourceLockSHA256 || original.Digest == next.Digest {
		t.Fatal("research-only semantic package lock and representation lineage conflated")
	}
}
