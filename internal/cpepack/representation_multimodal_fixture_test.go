package cpepack

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// Both payloads are original tiny synthetic bytes, embedded here solely to
// generate a disposable external-file fixture. This is not Foundry hydration.
const (
	motionFixture = "{\"schema\":\"fixture.motion.v0\",\"samples\":[{\"ms\":0,\"root\":[0,0,0]},{\"ms\":100,\"root\":[0,0.1,0]}]}\n"
	kmlFixture    = "<?xml version=\"1.0\" encoding=\"UTF-8\"?><kml xmlns=\"http://www.opengis.net/kml/2.2\"><Placemark><Point><coordinates>0,0,0</coordinates></Point></Placemark></kml>\n"
	motionSHA     = "a9a8e1bd962a5d839cbeac483428351815291c0a1fa6b49e727ea5f9099e34cd"
	kmlSHA        = "50ef9f0231d503cc0adbcc4d48e929b35429422d38a963fe02593915f7a08b1a"
)

func fixtureFileSHA256(t *testing.T, path string) string {
	t.Helper()
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(bytes)
	return hex.EncodeToString(sum[:])
}

func TestOriginalMotionKMLExternalFileSubsetReplay(t *testing.T) {
	walk := Concept{ID: "fixture:walk", Definition: "original synthetic stepwise motion", Relations: []Relation{{Predicate: "performed_at", TargetID: "fixture:place"}}}
	place := Concept{ID: "fixture:place", Definition: "original synthetic geographic point"}
	p := fixture("fixture.public.motion_kml", "public", nil, walk, place)
	lock := lockFor(p)
	sel := SubsetSelector{SchemaVersion: SubsetVersion, ProfileID: "fixture.motion.kml", RootConceptIDs: []string{"fixture:walk"}, RequiredEdges: []RequiredEdge{{FromID: "fixture:walk", Predicate: "performed_at", TargetID: "fixture:place"}}}
	refs := []RepresentationRef{
		{SchemaVersion: RepresentationVersion, ReferenceID: "fixture:motion-ref", ConceptID: "fixture:walk", Role: "demonstrates", Modality: "motion", MediaType: "application/json", FormatVersion: "fixture.motion.v0", Artifact: ArtifactIdentity{Kind: "raw-sha256", Digest: "sha256:" + motionSHA}, Visibility: "public", Rights: "redistributable", SourceRevision: "fixture-rev-1"},
		{SchemaVersion: RepresentationVersion, ReferenceID: "fixture:kml-ref", ConceptID: "fixture:place", Role: "parameterizes", Modality: "geospatial", MediaType: "application/vnd.google-earth.kml+xml", FormatVersion: "kml.2.2", Artifact: ArtifactIdentity{Kind: "raw-sha256", Digest: "sha256:" + kmlSHA}, Visibility: "public", Rights: "redistributable", SourceRevision: "fixture-rev-1"},
	}
	view, err := BuildSubset([]Package{p}, lock, []string{p.PackageID}, sel, refs, true)
	if err != nil || view.Authority != "research_only" || len(view.Concepts) != 2 || len(view.Representations) != 2 {
		t.Fatalf("qualified synthetic subset failed: %v %+v", err, view)
	}
	// The files are written into a disposable test directory, outside the CPE
	// semantic package. Their bytes are independently checked against constants.
	temp := t.TempDir()
	local := map[string]struct{ name, contents, expected string }{
		"fixture:motion-ref": {"walk.motion.json", motionFixture, motionSHA},
		"fixture:kml-ref":    {"place.kml", kmlFixture, kmlSHA},
	}
	for _, ref := range view.Representations {
		f, ok := local[ref.ReferenceID]
		if !ok || ref.DependencyOnly || ref.Artifact.Kind != "raw-sha256" {
			t.Fatalf("unexpected local reference: %+v", ref)
		}
		path := filepath.Join(temp, f.name)
		if err := os.WriteFile(path, []byte(f.contents), 0o600); err != nil {
			t.Fatal(err)
		}
		if actual := fixtureFileSHA256(t, path); actual != f.expected || ref.Artifact.Digest != "sha256:"+actual {
			t.Fatalf("local-byte identity mismatch: ref=%s got=%s", ref.ReferenceID, actual)
		}
	}
	// Reference input order must not cause a different selection or release view.
	refs[0], refs[1] = refs[1], refs[0]
	replayed, err := BuildSubset([]Package{p}, lock, []string{p.PackageID}, sel, refs, true)
	if err != nil || replayed.Digest != view.Digest {
		t.Fatalf("nondeterministic selected release: %v", err)
	}
	// Changed actual bytes must no longer match a previously selected raw hash.
	motionPath := filepath.Join(temp, "walk.motion.json")
	if err := os.WriteFile(motionPath, []byte("tampered fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if fixtureFileSHA256(t, motionPath) == motionSHA {
		t.Fatal("tampered fixture unexpectedly matched pinned SHA")
	}
	// Updating semantic meaning under the original source lock must fail.
	changed := p
	changed.Concepts = append([]Concept(nil), p.Concepts...)
	changed.Concepts[0].Definition = "unpromoted different semantic meaning"
	if _, err := BuildSubset([]Package{changed}, lock, []string{p.PackageID}, sel, refs, true); err == nil || err.Error() != "content_digest_mismatch" {
		t.Fatalf("semantic source mutation admitted under old lock: %v", err)
	}
}
