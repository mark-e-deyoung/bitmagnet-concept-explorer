package main

import (
	"strings"
	"testing"
)

func TestProposalDeterministic(t *testing.T) {
	ref := "sha256:" + strings.Repeat("a", 64)
	a, e := makeProposal("coverage_gap", "fictional", []string{ref})
	if e != nil {
		t.Fatal(e)
	}
	b, e := makeProposal("coverage_gap", "fictional", []string{ref})
	if e != nil {
		t.Fatal(e)
	}
	if a.ProposalID != b.ProposalID || a.State != "local-export-only" || a.ConceptID != nil {
		t.Fatalf("invalid proposal %#v", a)
	}
}
func TestProposalRejectsRawEvidence(t *testing.T) {
	if _, e := makeProposal("lexicalization_candidate", "fictional", []string{"torrent-name"}); e == nil {
		t.Fatal("raw evidence accepted")
	}
}
func TestLoopbackEnforcement(t *testing.T) {
	for _, v := range []string{"127.0.0.1:3086", "[::1]:3086", "localhost:3086"} {
		if !isLoopbackBind(v) {
			t.Fatalf("loopback rejected: %s", v)
		}
	}
	for _, v := range []string{":3086", "0.0.0.0:3086", "192.168.1.2:3086"} {
		if isLoopbackBind(v) {
			t.Fatalf("unsafe bind accepted: %s", v)
		}
	}
}
