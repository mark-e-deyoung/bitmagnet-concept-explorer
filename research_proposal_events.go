package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ResearchProposalEvent is an offline, append-only RESEARCH rehearsal record.
// A correct chain does not prove independent identity, accepted CPE authority,
// or permission to publish the source evidence.
type ResearchProposalEvent struct {
	SchemaVersion  string `json:"schema_version"`
	ProposalID     string `json:"proposal_id"`
	Sequence       int    `json:"sequence"`
	FromState      string `json:"from_state"`
	ToState        string `json:"to_state"`
	ActorKind      string `json:"actor_kind"`
	ActorRef       string `json:"actor_ref"`
	EvidenceDigest string `json:"evidence_digest"`
	PreviousHash   string `json:"previous_hash"`
	EventHash      string `json:"event_hash"`
}

var researchTransitions = map[string]map[string]bool{
	"local-export-only":  {"submitted-research": true},
	"submitted-research": {"classified": true, "deferred": true},
	"classified":         {"grounded": true, "deferred": true},
	"grounded":           {"challenged": true, "deferred": true},
	"challenged":         {"review-ready": true, "deferred": true},
	"review-ready":       {"recommend-mapping": true, "reject-candidate": true, "deferred": true},
}

func validResearchDigest(s string) bool {
	if !strings.HasPrefix(s, "sha256:") || len(s) != 71 {
		return false
	}
	_, err := hex.DecodeString(s[7:])
	return err == nil && s == strings.ToLower(s)
}

func researchEventHash(e ResearchProposalEvent) (string, error) {
	e.EventHash = ""
	serialized, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(serialized)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func replayResearchEvents(p DiscoveryProposal, chain []ResearchProposalEvent) (string, error) {
	if p.SchemaVersion != "research-v0" || p.State != "local-export-only" ||
		!strings.HasPrefix(p.ProposalID, "bce-") || len(p.ProposalID) != 68 {
		return "", errors.New("invalid research proposal identity or authority")
	}
	state := p.State
	prev := ""
	for i, e := range chain {
		if e.SchemaVersion != "research-proposal-event-v0" ||
			e.ProposalID != p.ProposalID || e.Sequence != i+1 ||
			e.FromState != state || !researchTransitions[state][e.ToState] ||
			e.PreviousHash != prev || e.ActorRef == "" || len(e.ActorRef) > 128 ||
			!validResearchDigest(e.EvidenceDigest) || !validResearchDigest(e.EventHash) {
			return "", fmt.Errorf("invalid research event at index %d", i)
		}
		if e.ToState == "recommend-mapping" || e.ToState == "reject-candidate" ||
			e.ToState == "deferred" {
			if e.ActorKind != "independent-reviewer" {
				return "", errors.New("terminal research disposition requires reviewer role")
			}
		} else if e.ActorKind != "researcher" && e.ActorKind != "machine-validator" {
			return "", errors.New("invalid research actor role")
		}
		digest, err := researchEventHash(e)
		if err != nil || digest != e.EventHash {
			return "", fmt.Errorf("research event digest mismatch at index %d", i)
		}
		prev = e.EventHash
		state = e.ToState
	}
	return state, nil
}

func appendResearchEvent(p DiscoveryProposal, chain []ResearchProposalEvent, to, actorKind, actorRef, evidenceDigest string) ([]ResearchProposalEvent, error) {
	current, err := replayResearchEvents(p, chain)
	if err != nil {
		return nil, err
	}
	if !researchTransitions[current][to] || actorRef == "" ||
		!validResearchDigest(evidenceDigest) {
		return nil, errors.New("invalid research state transition or evidence")
	}
	prev := ""
	if len(chain) > 0 {
		prev = chain[len(chain)-1].EventHash
	}
	e := ResearchProposalEvent{
		SchemaVersion:  "research-proposal-event-v0",
		ProposalID:     p.ProposalID,
		Sequence:       len(chain) + 1,
		FromState:      current,
		ToState:        to,
		ActorKind:      actorKind,
		ActorRef:       actorRef,
		EvidenceDigest: evidenceDigest,
		PreviousHash:   prev,
	}
	e.EventHash, err = researchEventHash(e)
	if err != nil {
		return nil, err
	}
	candidate := append(append([]ResearchProposalEvent(nil), chain...), e)
	if _, err := replayResearchEvents(p, candidate); err != nil {
		return nil, err
	}
	return candidate, nil
}
