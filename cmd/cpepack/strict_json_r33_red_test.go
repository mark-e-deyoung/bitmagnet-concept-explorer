package main

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/mark-e-deyoung/bitmagnet-concept-explorer/internal/cpepack"
)

// R33 frozen, independently authored negative controls against the existing
// offline JSON reader. Synthetic inputs only. Expected to FAIL on R32 parent.
// Encoding/json DisallowUnknownFields is not strict duplicate-key admission.
func r33Write(t *testing.T, data []byte) string {
    t.Helper()
    p := filepath.Join(t.TempDir(), "input.json")
    if err := os.WriteFile(p, data, 0600); err != nil { t.Fatal(err) }
    return p
}

func TestR33REDDuplicateTopLevelLockKeyIsRejected(t *testing.T) {
    raw := []byte(`{"schema_version":"cpe-lock-v0","schema_version":"cpe-lock-v0","packages":[]}`)
    var target cpepack.Lock
    if err := read(r33Write(t,raw),&target); err == nil {
        t.Fatal("R33 RED: duplicate top-level JSON member was silently accepted; last-write-wins ambiguity is not admissible for a pinned knowledge lock")
    }
}

func TestR33REDNestedDuplicatePackageIdentityIsRejected(t *testing.T) {
    raw := []byte(`{"schema_version":"cpe-lock-v0","packages":[{"package_id":"synthetic.first","package_id":"synthetic.second"}]}`)
    var target cpepack.Lock
    if err := read(r33Write(t,raw),&target); err == nil {
        t.Fatal("R33 RED: duplicate nested package ID was silently accepted; a source witness must not depend on decoder overwrite order")
    }
}

func TestR33REDMalformedUTF8InKnownJSONFieldIsRejected(t *testing.T) {
    raw := append([]byte(`{"schema_version":"`), 0xff)
    raw = append(raw, []byte(`","packages":[]}`)...)
    var target cpepack.Lock
    if err := read(r33Write(t,raw),&target); err == nil {
        t.Fatal("R33 RED: invalid UTF-8 was silently replaced by decoder; exact source identity and provenance must preserve or refuse malformed bytes")
    }
}
