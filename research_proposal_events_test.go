package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestResearchProposalEventReplayAndIndependentDisposition(t *testing.T) {
	digest := "sha256:" + strings.Repeat("a", 64)
	p, err := makeProposal("lexicalization_candidate", "test-only-native-token", []string{digest})
	if err != nil {
		t.Fatal(err)
	}
	chain := []ResearchProposalEvent{}
	for _, state := range []string{"submitted-research", "classified", "grounded", "challenged", "review-ready"} {
		chain, err = appendResearchEvent(p, chain, state, "machine-validator", "synthetic-test-validator", digest)
		if err != nil {
			t.Fatalf("failed at %q: %v", state, err)
		}
	}
	if _, err := appendResearchEvent(p, chain, "recommend-mapping", "researcher", "same-as-producer", digest); err == nil {
		t.Fatal("research proposal producer allowed to self-recommend terminal result")
	}
	if _, err := appendResearchEvent(p, chain, "accepted", "independent-reviewer", "synthetic-reviewer", digest); err == nil {
		t.Fatal("research rehearsal must not promote accepted CPE authority")
	}
	original := append([]ResearchProposalEvent(nil), chain...)
	chain, err = appendResearchEvent(p, chain, "recommend-mapping", "independent-reviewer", "synthetic-reviewer", digest)
	if err != nil {
		t.Fatal(err)
	}
	state, err := replayResearchEvents(p, chain)
	if err != nil || state != "recommend-mapping" {
		t.Fatalf("invalid independent review chain: %s %v", state, err)
	}
	if !reflect.DeepEqual(original, chain[:len(original)]) {
		t.Fatal("append silently rewrote prior research events")
	}
	clone := append([]ResearchProposalEvent(nil), chain...)
	clone[1].ActorRef = "tampered-actor"
	if _, err := replayResearchEvents(p, clone); err == nil {
		t.Fatal("tampered event not rejected")
	}
	clone = append([]ResearchProposalEvent(nil), chain...)
	clone[2].PreviousHash = digest
	if _, err := replayResearchEvents(p, clone); err == nil {
		t.Fatal("broken chain not rejected")
	}
	clone = append([]ResearchProposalEvent(nil), chain...)
	clone[4].Sequence = 77
	if _, err := replayResearchEvents(p, clone); err == nil {
		t.Fatal("sequence gap not rejected")
	}
}

func TestResearchProposalEventDeterminismAndMissingEvidence(t *testing.T) {
	ref := "sha256:" + strings.Repeat("b", 64)
	p, err := makeProposal("coverage_gap", "fixture", []string{ref})
	if err != nil {
		t.Fatal(err)
	}
	x, err := appendResearchEvent(p, nil, "submitted-research", "researcher", "synthetic-producer", ref)
	if err != nil {
		t.Fatal(err)
	}
	y, err := appendResearchEvent(p, nil, "submitted-research", "researcher", "synthetic-producer", ref)
	if err != nil || !reflect.DeepEqual(x, y) {
		t.Fatal("same research input did not yield same event/hash")
	}
	if _, err := appendResearchEvent(p, nil, "classified", "machine-validator", "synthetic-validator", ref); err == nil {
		t.Fatal("skipped research stage")
	}
	if _, err := appendResearchEvent(p, nil, "submitted-research", "researcher", "synthetic-producer", "not-a-digest"); err == nil {
		t.Fatal("unsigned/unattributed event evidence accepted")
	}
	if _, err := appendResearchEvent(p, x, "submitted-research", "researcher", "synthetic-producer", ref); err == nil {
		t.Fatal("duplicate state transition")
	}
}
