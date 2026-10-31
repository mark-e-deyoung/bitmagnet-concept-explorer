package cpepack

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestBCEPreviewDeterministicAndPrivateClosed(t *testing.T) {
	ps, lock := original()
	a, d, err := BuildBCEPreview(ps, lock, []string{"test.public.literature"})
	if err != nil {
		t.Fatal(err)
	}
	b, d2, err := BuildBCEPreview([]Package{ps[2], ps[0], ps[1]}, Lock{SchemaVersion: lock.SchemaVersion, Packages: []LockedPackage{lock.Packages[1], lock.Packages[2], lock.Packages[0]}}, []string{"test.public.literature"})
	if err != nil || !bytes.Equal(a, b) || d != d2 {
		t.Fatalf("non-deterministic BCE preview: %v", err)
	}
	if bytes.Contains(a, []byte("private:")) || bytes.Contains(a, []byte("test.private")) {
		t.Fatal("private material leaked")
	}
	if _, _, err = BuildBCEPreview(ps, lock, []string{"test.private.domain"}); err == nil {
		t.Fatal("private root emitted")
	}
	var v BCEPreview
	if err = json.Unmarshal(a, &v); err != nil {
		t.Fatal(err)
	}
	if v.SchemaVersion != "bce-graph-preview-v1" || len(v.Concepts) != 2 || !strings.HasPrefix(v.SnapshotID, "cpe-lock-sha256:") {
		t.Fatalf("bad graph: %+v", v)
	}
	if v.Concepts[0].Label != "work" || v.Concepts[1].Label != "work" {
		t.Fatal("English lexicalized label unstable")
	}
}
func TestBCEPreviewRequiresLexeme(t *testing.T) {
	p, l := original()
	p[0].Concepts[0].Lexicalizations = nil
	p[0].ContentSHA256, _ = ContentDigest(p[0].Concepts)
	l.Packages[0].ContentSHA256 = p[0].ContentSHA256
	if _, _, err := BuildBCEPreview(p, l, []string{"test.public.literature"}); err == nil || err.Error() != "missing_bce_display_label" {
		t.Fatalf("expected missing label; got %v", err)
	}
}
