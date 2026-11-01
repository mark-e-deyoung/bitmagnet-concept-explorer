package cpepack

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Test checked-in, independently replayable public-safe input bytes, not only constructors.
func TestCheckedInPackageLockReplay(t *testing.T) {
	path := func(name string) string { return filepath.Join("..", "..", "testdata", "cpepack", name) }
	rawPackages, err := os.ReadFile(path("packages.json"))
	if err != nil {
		t.Fatal(err)
	}
	rawLock, err := os.ReadFile(path("lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile(path("expected-public-projection.json"))
	if err != nil {
		t.Fatal(err)
	}
	var packages []Package
	if err := json.Unmarshal(rawPackages, &packages); err != nil {
		t.Fatal(err)
	}
	var lock Lock
	if err := json.Unmarshal(rawLock, &lock); err != nil {
		t.Fatal(err)
	}
	wantPublic, err := Build(packages, lock, []string{"test.public.literature"}, true)
	if err != nil {
		t.Fatal(err)
	}
	wantJSON, err := json.MarshalIndent(wantPublic, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	wantJSON = append(wantJSON, '\n')
	if !bytes.Equal(wantJSON, expected) {
		t.Fatalf("frozen public projection changed: expected %q, got %q", expected, wantJSON)
	}
	// The unrelated third private research package is never an implicit input to a public projection.
	gotWithoutPrivate, err := Build(packages[1:], Lock{SchemaVersion: lock.SchemaVersion, Packages: lock.Packages[1:]}, []string{"test.public.literature"}, true)
	if err != nil {
		t.Fatal(err)
	}
	otherJSON, _ := json.MarshalIndent(gotWithoutPrivate, "", "  ")
	otherJSON = append(otherJSON, '\n')
	if !bytes.Equal(expected, otherJSON) {
		t.Fatal("unrelated synthetic private package influenced public digest")
	}
	if _, err := Build(packages, lock, []string{"test.private.domain"}, true); err == nil || err.Error() != "private_dependency_in_public_closure" {
		t.Fatalf("private closure not refused: %v", err)
	}
}
