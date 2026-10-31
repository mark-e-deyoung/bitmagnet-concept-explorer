package main

import (
	"strings"
	"testing"
)

func warningCount(warnings []string, target string) int {
	count := 0
	for _, warning := range warnings {
		if warning == target {
			count++
		}
	}
	return count
}

func TestPinnedFTSMultiwordQueryPlanWarnsWithoutRewriting(t *testing.T) {
	plan, err := makePlan(Request{
		Query: "fiction | !alternate",
		Expansions: []Expansion{{
			Term: "science fiction", ConceptID: "media.test",
			Provenance: "digest-pinned-preview:synthetic",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Tracks[0].Terms[0] != "fiction | !alternate" || len(plan.Tracks[1].Terms) != 1 || plan.Tracks[1].Terms[0] != "science fiction" {
		t.Fatalf("altered native FTS or pinned candidate: %#v", plan.Tracks)
	}
	if warningCount(plan.Warnings, pinnedMultiwordFTSWarning) != 1 {
		t.Fatalf("unquoted generated multiword form lacks exact parser warning: %v", plan.Warnings)
	}
	for _, tc := range []struct {
		name, term, provenance string
	}{
		{"pinned-single-word", "fiction", "digest-pinned-preview:synthetic"},
		{"manual-multiword", "science fiction", "manual-unverified"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, err := makePlan(Request{Query: "animation", Expansions: []Expansion{{Term: tc.term, Provenance: tc.provenance}}})
			if err != nil {
				t.Fatal(err)
			}
			if warningCount(p.Warnings, pinnedMultiwordFTSWarning) != 0 {
				t.Fatalf("unexpected generated-multiword warning for %s: %v", tc.name, p.Warnings)
			}
		})
	}
}

func TestPinnedFTSMultiwordSweepWarnsOnceAndPreservesUnknown(t *testing.T) {
	setupFixtureGraph(t, sweepFixture)
	plan, err := makeSweepPlan(SweepRequest{
		Query: "fiction | !alternate",
		ConceptIDs: []string{"media.animation"},
		Languages: []string{"en"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Bitmagnet) < 2 || plan.Bitmagnet[0].Term != "fiction | !alternate" {
		t.Fatalf("direct query altered or synthetic graph absent: %#v", plan.Bitmagnet)
	}
	var sawMultiword bool
	for _, term := range plan.Bitmagnet {
		if term.Term == "animated film" && strings.HasPrefix(term.Provenance, "pinned-preview:") {
			sawMultiword = true
		}
	}
	if !sawMultiword || warningCount(plan.Warnings, pinnedMultiwordFTSWarning) != 1 {
		t.Fatalf("missing multiword sweep warning or query: %#v", plan)
	}
	if plan.Authority != "digest-pinned-preview-not-cpe-accepted" || !plan.ReadOnly || plan.ExternalAdmitted {
		t.Fatalf("changed research-only/admission boundaries: %#v", plan)
	}
}
