package cpepack

import "testing"

// The source of expected digests is synthetically constructed in these tests
// to exercise the mechanics. A real consumer MUST obtain them from independently
// authorized/pinned release authority, not a self-declared candidate receipt.
func TestPinnedSubsetAdmitsExactResearchReferenceSelection(t *testing.T) {
	ps, lock, sel, refs := redPinnedSourceFixture()
	unpinned, err := BuildSubset(ps, lock, []string{ps[0].PackageID}, sel, refs, true)
	if err != nil {
		t.Fatal(err)
	}
	got, err := BuildPinnedSubset(ps, lock, []string{ps[0].PackageID}, sel, refs, true, unpinned.SourceLockSHA256, unpinned.Digest)
	if err != nil || got.Digest != unpinned.Digest || got.Authority != "research_only" {
		t.Fatalf("exact research pin rejected or authority upgraded: view=%+v err=%v", got, err)
	}
}

func TestPinnedSubsetRefusesMissingOrMalformedIndependentExpectation(t *testing.T) {
	ps, lock, sel, refs := redPinnedSourceFixture()
	for _, test := range []struct{ source, selection string }{
		{"", ""},
		{"abcd", fixtureHash},
		{fixtureHash, "deadbeef"},
		{fixtureHash, ""},
	} {
		if _, err := BuildPinnedSubset(ps, lock, []string{ps[0].PackageID}, sel, refs, true, test.source, test.selection); err == nil || err.Error() != "missing_or_invalid_independent_pin" {
			t.Fatalf("weak pin accepted: source=%q selection=%q err=%v", test.source, test.selection, err)
		}
	}
}

func TestPinnedSubsetRefusesChangedExampleUnderSameSemanticLock(t *testing.T) {
	ps, lock, sel, refs := redPinnedSourceFixture()
	expected, err := BuildSubset(ps, lock, []string{ps[0].PackageID}, sel, refs, true)
	if err != nil {
		t.Fatal(err)
	}
	refs[0].Artifact.Digest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	_, err = BuildPinnedSubset(ps, lock, []string{ps[0].PackageID}, sel, refs, true, expected.SourceLockSHA256, expected.Digest)
	if err == nil || err.Error() != "selection_pin_mismatch" {
		t.Fatalf("changed external example was selected under old pin: %v", err)
	}
}

func TestPinnedSubsetRefusesChangedSourceRevisionAndWrongRootLock(t *testing.T) {
	ps, lock, sel, refs := redPinnedSourceFixture()
	expected, err := BuildSubset(ps, lock, []string{ps[0].PackageID}, sel, refs, true)
	if err != nil {
		t.Fatal(err)
	}
	refs[0].SourceRevision = "fixture-src-v2"
	_, err = BuildPinnedSubset(ps, lock, []string{ps[0].PackageID}, sel, refs, true, expected.SourceLockSHA256, expected.Digest)
	if err == nil || err.Error() != "selection_pin_mismatch" {
		t.Fatalf("mutated source lineage accepted: %v", err)
	}
	refs[0].SourceRevision = "fixture-src-v1"
	_, err = BuildPinnedSubset(ps, lock, []string{ps[0].PackageID}, sel, refs, true, fixtureHash, expected.Digest)
	if err == nil || err.Error() != "source_pin_mismatch" {
		t.Fatalf("wrong source lock accepted: %v", err)
	}
}

func TestPinnedSubsetCannotBypassPrivateClosureOrSemanticSourceGuard(t *testing.T) {
	ps, lock, sel, refs := redPinnedSourceFixture()
	expected, err := BuildSubset(ps, lock, []string{ps[0].PackageID}, sel, refs, true)
	if err != nil {
		t.Fatal(err)
	}
	refs[0].Visibility = "private"
	_, err = BuildPinnedSubset(ps, lock, []string{ps[0].PackageID}, sel, refs, true, expected.SourceLockSHA256, expected.Digest)
	if err == nil || err.Error() != "private_representation_in_public_closure" {
		t.Fatalf("private representation escaped admission: %v", err)
	}
	refs[0].Visibility = "public"
	ps[0].Concepts[0].Definition = "unapproved changed meaning"
	_, err = BuildPinnedSubset(ps, lock, []string{ps[0].PackageID}, sel, refs, true, expected.SourceLockSHA256, expected.Digest)
	if err == nil || err.Error() != "content_digest_mismatch" {
		t.Fatalf("altered semantic source bypassed lock: %v", err)
	}
}
