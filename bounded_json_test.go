package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestDecodeBoundedJSONRejectsValidPrefixWithOversizedTrailingPadding(t *testing.T) {
	for _, body := range []string{
		`{"x":1}` + strings.Repeat(" ", 300),
		`{"x":1}` + strings.Repeat("\n", 300),
		`{"x":1}{"x":2}`,
		`{"x":1}!`,
	} {
		var out map[string]any
		if err := decodeBoundedJSON(strings.NewReader(body), 128, &out); err == nil {
			t.Fatalf("accepted trailing/oversized upstream JSON: length %d", len(body))
		}
	}
}
func TestDecodeBoundedJSONExactBudgetAndValidOneDocument(t *testing.T) {
	raw := []byte(`{"x":1}`)
	var out map[string]any
	if e := decodeBoundedJSON(bytes.NewReader(raw), len(raw), &out); e != nil {
		t.Fatalf("valid exact-sized JSON rejected: %v", e)
	}
	if out["x"] != float64(1) {
		t.Fatalf("incorrect JSON result %#v", out)
	}
	for _, max := range []int{0, len(raw) - 1} {
		if e := decodeBoundedJSON(bytes.NewReader(raw), max, &out); e == nil {
			t.Fatalf("invalid byte budget %d accepted", max)
		}
	}
}
