package cpepack

import (
	"encoding/json"
	"strings"
	"testing"
)

const fixtureHash = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func sampleSource() Projection {
	return Projection{SchemaVersion: SnapshotVersion, Authority: "research_only", LockSHA256: fixtureHash, Concepts: []Concept{
		{ID: "fixture:walk", Definition: "walking movement", Relations: []Relation{{Predicate: "requires", TargetID: "fixture:steps"}, {Predicate: "related", TargetID: "fixture:voice"}}},
		{ID: "fixture:steps", Definition: "step cycle"},
		{ID: "fixture:voice", Definition: "voice characteristics"},
		{ID: "fixture:dance", Definition: "dance", Relations: []Relation{{Predicate: "requires", TargetID: "fixture:walk"}}},
	}}
}
func sampleSelector(ids ...string) SubsetSelector {
	return SubsetSelector{SchemaVersion: SubsetVersion, ProfileID: "fixture.motion", RootConceptIDs: ids, RequiredEdges: []RequiredEdge{{"fixture:walk", "requires", "fixture:steps"}, {"fixture:dance", "requires", "fixture:walk"}}}
}
func sampleRefs() []RepresentationRef {
	newRef := func(id, concept, kind, role, modality, visibility, rights string) RepresentationRef {
		return RepresentationRef{SchemaVersion: RepresentationVersion, ReferenceID: id, ConceptID: concept, Role: role, Modality: modality, MediaType: "application/json", FormatVersion: "fixture.v1", Artifact: ArtifactIdentity{Kind: kind, Digest: "sha256:" + fixtureHash}, Visibility: visibility, Rights: rights, SourceRevision: "fixture-rev-1"}
	}
	return []RepresentationRef{
		newRef("fixture:ref-walk", "fixture:walk", "raw-sha256", "demonstrates", "motion", "public", "redistributable"),
		newRef("fixture:ref-steps", "fixture:steps", "content-manifest", "parameterizes", "motion", "public", "redistributable"),
		newRef("fixture:ref-voice", "fixture:voice", "oci", "demonstrates", "voice", "private", "restricted"),
		newRef("fixture:ref-dance", "fixture:dance", "raw-sha256", "demonstrates", "motion", "public", "unknown"),
	}
}
func buildTest(t *testing.T, selector SubsetSelector, refs []RepresentationRef, public bool) (SubsetView, error) {
	t.Helper()
	return MaterializeSubset(sampleSource(), selector, refs, public)
}
func expectErr(t *testing.T, want string, selector SubsetSelector, refs []RepresentationRef, public bool) {
	t.Helper()
	_, err := buildTest(t, selector, refs, public)
	if err == nil || err.Error() != want {
		t.Fatalf("want %s, got %v", want, err)
	}
}
func TestSubsetFixtureRequiredAndExternalEdges(t *testing.T) {
	out, e := buildTest(t, sampleSelector("fixture:walk"), sampleRefs(), true)
	if e != nil {
		t.Fatal(e)
	}
	if len(out.Concepts) != 2 || out.Concepts[0].ID != "fixture:steps" || out.Concepts[1].ID != "fixture:walk" {
		t.Fatalf("wrong closure: %+v", out.Concepts)
	}
	if len(out.OutOfSubsetEdges) != 1 || out.OutOfSubsetEdges[0].TargetID != "fixture:voice" {
		t.Fatalf("external edge lost: %+v", out.OutOfSubsetEdges)
	}
	if len(out.Representations) != 2 || out.Authority != "research_only" || out.SourceLockSHA256 != fixtureHash || len(out.Digest) != 64 {
		t.Fatalf("output invalid %+v", out)
	}
	for _, r := range out.Representations {
		if r.Visibility == "private" {
			t.Fatal("leaked private reference")
		}
	}
}
func TestReferenceInputOrderAndSelectorOrderInvariant(t *testing.T) {
	s := sampleSelector("fixture:walk", "fixture:dance")
	a, e := buildTest(t, s, sampleRefs(), true)
	if e != nil {
		t.Fatal(e)
	}
	rs := sampleRefs()
	for i, j := 0, len(rs)-1; i < j; i, j = i+1, j-1 {
		rs[i], rs[j] = rs[j], rs[i]
	}
	s.RootConceptIDs = []string{"fixture:dance", "fixture:walk"}
	s.RequiredEdges[0], s.RequiredEdges[1] = s.RequiredEdges[1], s.RequiredEdges[0]
	b, e := buildTest(t, s, rs, true)
	if e != nil {
		t.Fatal(e)
	}
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	if string(x) != string(y) {
		t.Fatalf("permutation changes view: %s vs %s", x, y)
	}
}
func TestDifferentSubsetChangesDigestAndNoPrivateLeak(t *testing.T) {
	a, e := buildTest(t, sampleSelector("fixture:walk"), sampleRefs(), true)
	if e != nil {
		t.Fatal(e)
	}
	b, e := buildTest(t, sampleSelector("fixture:dance"), sampleRefs(), true)
	if e != nil {
		t.Fatal(e)
	}
	if a.Digest == b.Digest || len(b.Concepts) != 3 {
		t.Fatal("subset not distinct")
	}
	raw, _ := json.Marshal(a)
	if strings.Contains(string(raw), "fixture:ref-voice") {
		t.Fatal("private optional ref leaked")
	}
}
func TestSameBytesDistinctSemanticRolesPreserved(t *testing.T) {
	refs := sampleRefs()
	refs[1].Artifact = refs[0].Artifact
	out, e := buildTest(t, sampleSelector("fixture:walk"), refs, true)
	if e != nil {
		t.Fatal(e)
	}
	if len(out.Representations) != 2 || out.Representations[0].ConceptID == out.Representations[1].ConceptID {
		t.Fatal("references deduplicated by artifact digest")
	}
}
func TestMissingMediaDoesNotDeleteConcept(t *testing.T) {
	out, e := buildTest(t, sampleSelector("fixture:walk"), nil, true)
	if e != nil {
		t.Fatal(e)
	}
	if len(out.Concepts) != 2 || len(out.Representations) != 0 {
		t.Fatal("missing examples changed semantic closure")
	}
}
func TestRequiredReferenceClosureAndDependencyOnly(t *testing.T) {
	refs := sampleRefs()
	refs[0].Dependencies = []string{"fixture:ref-dance"}
	out, e := buildTest(t, sampleSelector("fixture:walk"), refs, true)
	if e != nil {
		t.Fatal(e)
	}
	if len(out.Representations) != 3 {
		t.Fatalf("missing artifact dep %+v", out.Representations)
	}
	found := false
	for _, r := range out.Representations {
		if r.ReferenceID == "fixture:ref-dance" {
			found = r.DependencyOnly
		}
	}
	if !found {
		t.Fatal("dependency-only reference not marked")
	}
}
func TestPublicPrivateAndRightsDependencyNegative(t *testing.T) {
	t.Run("selected private", func(t *testing.T) {
		expectErr(t, "private_representation_in_public_closure", sampleSelector("fixture:voice"), sampleRefs(), true)
	})
	t.Run("transitive private", func(t *testing.T) {
		refs := sampleRefs()
		refs[0].Dependencies = []string{"fixture:ref-voice"}
		expectErr(t, "private_representation_in_public_closure", sampleSelector("fixture:walk"), refs, true)
	})
	t.Run("restricted public", func(t *testing.T) {
		refs := sampleRefs()
		refs[0].Rights = "restricted"
		expectErr(t, "restricted_representation_in_public_closure", sampleSelector("fixture:walk"), refs, true)
	})
	t.Run("private local okay", func(t *testing.T) {
		out, e := buildTest(t, sampleSelector("fixture:voice"), sampleRefs(), false)
		if e != nil || len(out.Representations) != 1 {
			t.Fatalf("private read failed: %v", e)
		}
	})
}
func TestNegativeSelectorAndIdentityGuards(t *testing.T) {
	cases := []struct {
		name, want string
		mutate     func(*SubsetSelector, *[]RepresentationRef)
	}{
		{"bad selector version", "invalid_subset_selector", func(s *SubsetSelector, _ *[]RepresentationRef) { s.SchemaVersion = "v1" }},
		{"unknown root", "unknown_subset_root", func(s *SubsetSelector, _ *[]RepresentationRef) { s.RootConceptIDs = []string{"fixture:missing"} }},
		{"duplicate root", "invalid_subset_roots", func(s *SubsetSelector, _ *[]RepresentationRef) {
			s.RootConceptIDs = []string{"fixture:walk", "fixture:walk"}
		}},
		{"fabricated required edge", "unproven_required_edge", func(s *SubsetSelector, _ *[]RepresentationRef) {
			s.RequiredEdges = []RequiredEdge{{"fixture:walk", "requires", "fixture:voice"}}
		}},
		{"duplicate required edge", "duplicate_required_edge", func(s *SubsetSelector, _ *[]RepresentationRef) {
			s.RequiredEdges = append(s.RequiredEdges, s.RequiredEdges[0])
		}},
		{"bad kind", "invalid_artifact_identity_kind", func(_ *SubsetSelector, r *[]RepresentationRef) { (*r)[0].Artifact.Kind = "oci-impersonated" }},
		{"bad digest", "invalid_artifact_digest", func(_ *SubsetSelector, r *[]RepresentationRef) { (*r)[0].Artifact.Digest = "sha256:deadbeef" }},
		{"missing source", "incomplete_representation_metadata", func(_ *SubsetSelector, r *[]RepresentationRef) { (*r)[0].SourceRevision = "" }},
		{"duplicate ref", "duplicate_representation_id", func(_ *SubsetSelector, r *[]RepresentationRef) { *r = append(*r, (*r)[0]) }},
		{"missing artifact dep", "representation_dependency_missing", func(_ *SubsetSelector, r *[]RepresentationRef) { (*r)[0].Dependencies = []string{"fixture:missing"} }},
		{"unbound concept", "representation_unknown_concept", func(_ *SubsetSelector, r *[]RepresentationRef) { (*r)[0].ConceptID = "fixture:nonexistent" }},
		{"bad rights", "invalid_representation_rights", func(_ *SubsetSelector, r *[]RepresentationRef) { (*r)[0].Rights = "granted_by_concept" }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, r := sampleSelector("fixture:walk"), sampleRefs()
			c.mutate(&s, &r)
			expectErr(t, c.want, s, r, true)
		})
	}
}
func TestAuthorityAndLockGate(t *testing.T) {
	for _, mod := range []func(*Projection){func(p *Projection) { p.Authority = "accepted" }, func(p *Projection) { p.LockSHA256 = "" }, func(p *Projection) { p.SchemaVersion = "other" }} {
		p := sampleSource()
		mod(&p)
		_, err := MaterializeSubset(p, sampleSelector("fixture:walk"), nil, true)
		if err == nil || err.Error() != "unqualified_source_projection" {
			t.Fatalf("admitted bad authority/lock: %v", err)
		}
	}
}
func TestStableAncestorSourceLockSelection(t *testing.T) {
	orig := sampleSource()
	a, e := MaterializeSubset(orig, sampleSelector("fixture:walk"), sampleRefs(), true)
	if e != nil {
		t.Fatal(e)
	}
	newer := sampleSource()
	newer.LockSHA256 = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	newer.Concepts = append(newer.Concepts, Concept{ID: "fixture:new", Definition: "unrelated"})
	b, e := MaterializeSubset(newer, sampleSelector("fixture:walk"), sampleRefs(), true)
	if e != nil {
		t.Fatal(e)
	}
	if a.Digest == b.Digest {
		t.Fatal("changed source release lock must change view digest")
	}
	replay, e := MaterializeSubset(orig, sampleSelector("fixture:walk"), sampleRefs(), true)
	if e != nil || replay.Digest != a.Digest {
		t.Fatal("historical replay was altered")
	}
}
func TestArtifactDependencyCycleRejected(t *testing.T) {
	refs := sampleRefs()
	refs[0].Dependencies = []string{"fixture:ref-steps"}
	refs[1].Dependencies = []string{"fixture:ref-walk"}
	expectErr(t, "representation_dependency_cycle", sampleSelector("fixture:walk"), refs, true)
}
