package main

import (
    "strings"
    "testing"
)

// Synthetic, self-contained native property controls. The original R33 RED
// source is unchanged in the base branch and remains the decisive negative.
func TestR34DuplicateKeyWithEscapedEquivalentSpellingIsRejected(t *testing.T) {
    raw := []byte(`{"id":1,"\u0069d":2}`)
    if err := strictJSONAdmission(raw); err == nil || err.Error() != "duplicate_json_key" {
        t.Fatalf("lexically distinct escapes for the same JSON member were not rejected: %v", err)
    }
}

func TestR34SameMemberNameInDistinctObjectScopesAllowed(t *testing.T) {
    raw := []byte(`{"left":{"id":"one"},"right":{"id":"two"}}`)
    if err := strictJSONAdmission(raw); err != nil {
        t.Fatalf("legitimate independent object scopes were conflated: %v", err)
    }
}

func TestR34MultilingualValidUTF8Preserved(t *testing.T) {
    raw := []byte(`{"language":["en","de","th"],"term":["work","Werk","งาน"]}`)
    if err := strictJSONAdmission(raw); err != nil {
        t.Fatalf("valid English/German/Thai input incorrectly rejected: %v", err)
    }
}

func TestR34MalformedTrailingDocumentRefused(t *testing.T) {
    if err := strictJSONAdmission([]byte(`{"id":1}{"id":2}`)); err == nil {
        t.Fatal("multiple top-level JSON documents passed byte admission")
    }
}

func TestR34DepthBudgetFailsClosed(t *testing.T) {
    raw := []byte(strings.Repeat("[", r34MaxJSONDepth+2) + "0" + strings.Repeat("]",r34MaxJSONDepth+2))
    if err := strictJSONAdmission(raw); err == nil || err.Error() != "json_depth_limit" {
        t.Fatalf("overbudget nested JSON was accepted or misclassified: %v",err)
    }
}
