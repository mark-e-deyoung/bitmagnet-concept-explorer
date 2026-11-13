package cpepack

import (
	"encoding/json"
	"strings"
	"testing"
)

const redHex = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func redFixture(refs ...RepresentationRef) (SubsetView, error) {
	source := Projection{SchemaVersion: SnapshotVersion, Authority: "research_only", LockSHA256: redHex, Concepts: []Concept{
		{ID: "fixture:walk", Definition: "Original synthetic gait"},
		{ID: "fixture:kml", Definition: "Original synthetic coordinate example"},
	}}
	selector := SubsetSelector{SchemaVersion: SubsetVersion, ProfileID: "fixture.public", RootConceptIDs: []string{"fixture:walk"}}
	return materializeSubset(source, selector, refs, true)
}
func redRef() RepresentationRef {
	return RepresentationRef{
		SchemaVersion: RepresentationVersion, ReferenceID: "fixture:walking-bvh", ConceptID: "fixture:walk",
		Role: "demonstrates", Modality: "motion", MediaType: "application/json", FormatVersion: "fixture.v1",
		Artifact:   ArtifactIdentity{Kind: "raw-sha256", Digest: "sha256:" + redHex},
		Visibility: "public", Rights: "redistributable", SourceRevision: "fixture-rev-1",
	}
}

// Source revisions are identifiers, not places to hide credentials/hostnames.
// This catches privacy leakage to the public deterministic projection, even
// when the reference itself claims to be public and redistributable.
func TestREDPublicRejectsControlCharacterSourceRevision(t *testing.T) {
	r := redRef()
	r.SourceRevision = "rev-1\nAuthorization: Bearer synthetic-do-not-use"
	view, err := redFixture(r)
	if err == nil {
		b, _ := json.Marshal(view)
		t.Fatalf("PUBLIC_METADATA_LEAK: rejected control characters were exported: %q", b)
	}
}
func TestREDPublicRejectsURIInMetadataIdentifier(t *testing.T) {
	r := redRef()
	r.Modality = "https://example.invalid/private/opaque-reference"
	view, err := redFixture(r)
	if err == nil {
		b, _ := json.Marshal(view)
		t.Fatalf("PUBLIC_METADATA_LEAK: URI-shaped modality was exported: %q", b)
	}
}
func TestREDPublicRejectsOpaqueMIMEParameters(t *testing.T) {
	r := redRef()
	r.MediaType = `application/json; source="https://example.invalid/private/blob"`
	view, err := redFixture(r)
	if err == nil {
		b, _ := json.Marshal(view)
		t.Fatalf("PUBLIC_METADATA_LEAK: opaque MIME parameter exported: %q", b)
	}
}
func TestREDPublicRejectsUnboundedExternalMetadata(t *testing.T) {
	r := redRef()
	r.FormatVersion = strings.Repeat("X", 65536)
	if _, err := redFixture(r); err == nil {
		t.Fatal("UNBOUNDED_METADATA: 64KiB format_version admitted into a public projection")
	}
}
func TestREDCleanKMLAndRawIdentityRemainAdmitted(t *testing.T) {
	r := redRef()
	r.Role, r.Modality, r.MediaType, r.FormatVersion = "parameterizes", "geospatial", "application/vnd.google-earth.kml+xml", "kml.2.2"
	if _, err := redFixture(r); err != nil {
		t.Fatalf("false positive on ordinary media metadata: %v", err)
	}
}
func TestREDMissingExampleDoesNotEraseConcept(t *testing.T) {
	a, err := redFixture()
	if err != nil || len(a.Concepts) != 1 || len(a.Representations) != 0 {
		t.Fatalf("semantic projection affected by missing optional media: view=%+v err=%v", a, err)
	}
}
func TestREDTypedDigestNamespaceNotConflated(t *testing.T) {
	a := redRef()
	b := a
	b.Artifact.Kind = "content-manifest"
	av, errA := redFixture(a)
	bv, errB := redFixture(b)
	if errA != nil || errB != nil || av.Digest == bv.Digest {
		t.Fatalf("distinct artifact identity kinds conflated: a=%s b=%s errs=%v,%v", av.Digest, bv.Digest, errA, errB)
	}
}
