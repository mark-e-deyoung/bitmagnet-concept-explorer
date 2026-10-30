package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSovereignRouteBlockedWithoutLocalConfig(t *testing.T) {
	t.Setenv("BCE_SOVEREIGN_BASE_URL", "https://api.openai.com/v1")
	t.Setenv("BCE_SOVEREIGN_MODEL", "whatever")
	if _, e := sovereignRoute(); e == nil {
		t.Fatal("cloud route accepted")
	}
	t.Setenv("BCE_SOVEREIGN_BASE_URL", "http://8.8.8.8:8000/v1")
	if _, e := sovereignRoute(); e == nil {
		t.Fatal("public IP accepted")
	}
	t.Setenv("BCE_SOVEREIGN_BASE_URL", "http://internal.example:8000/v1")
	if _, e := sovereignRoute(); e == nil {
		t.Fatal("hostname accepted")
	}
}
func TestSovereignValidatedProposalAndNoTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path %s", r.URL.Path)
		}
		var req struct {
			Model          string `json:"model"`
			ResponseFormat struct {
				Type string `json:"type"`
			} `json:"response_format"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if e := json.NewDecoder(r.Body).Decode(&req); e != nil {
			t.Fatal(e)
		}
		if req.ResponseFormat.Type != "json_object" || len(req.Messages) != 2 || !strings.Contains(req.Messages[1].Content, "Cyberpunk") {
			t.Error("bad payload")
		}
		fmt.Fprint(w, `{"model":"local","choices":[{"finish_reason":"stop","message":{"content":"{\"candidates\":[{\"candidate_label\":\"Cyberpunk\",\"concept_id\":\"genre.cyberpunk\",\"evidence_field\":\"title\",\"evidence_excerpt\":\"Cyberpunk\",\"rationale\":\"Literal title evidence only\"}]}"}}]}`)
	}))
	defer server.Close()
	t.Setenv("BCE_SOVEREIGN_BASE_URL", server.URL+"/v1")
	t.Setenv("BCE_SOVEREIGN_MODEL", "fast")
	res, e := inferSovereign(context.Background(), SovereignInput{Title: "Cyberpunk game", CandidateConceptIDs: []string{"genre.cyberpunk"}})
	if e != nil {
		t.Fatal(e)
	}
	if res.Authority != "unverified_model_hypothesis_only" || len(res.Candidates) != 1 {
		t.Fatalf("bad result %+v", res)
	}
}
func TestSovereignBlocksInventedEvidenceAndConcept(t *testing.T) {
	for _, content := range []string{`{"candidates":[{"candidate_label":"fake","concept_id":"not-authorized","evidence_field":"title","evidence_excerpt":"Cyberpunk","rationale":"x"}]}`, `{"candidates":[{"candidate_label":"fake","evidence_field":"title","evidence_excerpt":"not-there","rationale":"x"}]}`} {
		t.Run(content[:20], func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				x, _ := json.Marshal(content)
				fmt.Fprintf(w, `{"choices":[{"finish_reason":"stop","message":{"content":%s}}]}`, x)
			}))
			defer server.Close()
			t.Setenv("BCE_SOVEREIGN_BASE_URL", server.URL+"/v1")
			t.Setenv("BCE_SOVEREIGN_MODEL", "local")
			_, e := inferSovereign(context.Background(), SovereignInput{Title: "Cyberpunk game", CandidateConceptIDs: []string{"genre.cyberpunk"}})
			if e == nil {
				t.Fatal("invented output accepted")
			}
		})
	}
}

func TestSovereignAdmissionFailClosedWhenOccupied(t *testing.T) {
	t.Setenv("BCE_SOVEREIGN_BASE_URL", "http://127.0.0.1:18080/v1")
	t.Setenv("BCE_SOVEREIGN_MODEL", "synthetic-capability")
	globalSovereignSlots <- struct{}{}
	defer func() { <-globalSovereignSlots }()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, e := inferSovereign(ctx, SovereignInput{Title: "fictional"})
	if e == nil || !strings.Contains(e.Error(), "admission cancelled") {
		t.Fatalf("expected bounded local-only admission failure: %v", e)
	}
}
