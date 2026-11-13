package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The first two MiB may contain a complete JSON object and only whitespace.
// A 2-MiB-only LimitReader would mistake its artificial EOF for the actual EOF.
func TestReadRejectsOversizedTrailingData(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "adversarial.json")
	payload := "{}" + strings.Repeat(" ", 2<<20) + "!"
	if err := os.WriteFile(f, []byte(payload), 0600); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := read(f, &got); err == nil {
		t.Fatal("oversized trailing data was silently ignored")
	}
}
