package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const historicalPlanFixture = `{
 "schema_version":"experimental-v0","execution_id":"fixture-adapter","search_plan_ref":"fixture",
 "target_capability_profile_ref":"bitmagnet-v0.10.0","status":"ready",
 "required_target_capabilities":["keyword_search","unicode_query","json_results"],
 "optional_target_capabilities":[],
 "tactical_ir":{"clauses":[{"clause_id":"a","concept_id":"media.production.animation","purpose":"candidate_retrieval","relation":"must",
 "forms":[{"value":"animation","language":"en","form_type":"preferred_term","provenance_ref":"fixture:en"},
 {"value":"Animation","language":"de","form_type":"alias","provenance_ref":"fixture:de"}]}]},
 "execution_constraints":{"mutation_allowed":false,"max_queries":2,"authenticated_execution_allowed":false},
 "privacy_effect":{"external_disclosure":"none","authentication":"none","cross_surface_identifier":false,
 "local_query_history":"none","target_retention":"unknown","correlation_risk":"low","residual_risks":[]},
 "observation_requirements":{"emit_normalized_observations":true,"preserve_target_coverage":true,
 "no_match_is_negative_evidence":false}
}`

func preflightFromText(t *testing.T, s string) CPEPreflightResult {
	t.Helper()
	var p HistoricalAdapterPlan
	if e := json.Unmarshal([]byte(s), &p); e != nil {
		t.Fatal(e)
	}
	result, e := preflightHistoricalCPEPlan(p)
	if e != nil {
		t.Fatal(e)
	}
	return result
}
func TestHistoricalCPEPreflightIsNotExecutionAuthority(t *testing.T) {
	p := preflightFromText(t, historicalPlanFixture)
	if p.Status != "structurally_compatible_not_authorized" || p.ExecutionAuthorized || p.CompatibleLexicalForms != 2 || p.SemanticAuthority != "not_accepted_cpe_projection" {
		t.Fatalf("unsafe CPE promotion %#v", p)
	}
}
func TestHistoricalCPEPreflightDoesNotDropPostchecksOrHardNegatives(t *testing.T) {
	for _, variant := range []string{
		strings.Replace(historicalPlanFixture, `"status":"ready"`, `"status":"ready_with_postchecks"`, 1),
		strings.Replace(historicalPlanFixture, `"relation":"must"`, `"relation":"hard_negative"`, 1),
		strings.Replace(historicalPlanFixture, `"external_disclosure":"none"`, `"external_disclosure":"semantic_query"`, 1),
		strings.Replace(historicalPlanFixture, `"mutation_allowed":false`, `"mutation_allowed":true`, 1),
		strings.Replace(historicalPlanFixture, `"max_queries":2`, `"max_queries":1`, 1),
	} {
		p := preflightFromText(t, variant)
		if p.Status != "blocked" || len(p.Reasons) == 0 || p.ExecutionAuthorized {
			t.Fatalf("unsafe plan admitted %#v", p)
		}
	}
}
func TestHistoricalCPERejectsPolicyContextInput(t *testing.T) {
	s := httptest.NewServer(handler())
	defer s.Close()
	inp := strings.Replace(historicalPlanFixture, `"schema_version":"experimental-v0"`, `"policy_context":{"secret":"not permitted"},"schema_version":"experimental-v0"`, 1)
	r, e := http.Post(s.URL+"/api/v1/cpe/adapter-plan/preflight", "application/json", bytes.NewBufferString(inp))
	if e != nil {
		t.Fatal(e)
	}
	defer r.Body.Close()
	if r.StatusCode != 400 {
		t.Fatalf("policy state was accepted: %d", r.StatusCode)
	}
}

func TestHistoricalCPEPreflightRefusesUnsupportedMultiClauseAndTag(t *testing.T) {
	multi := strings.Replace(historicalPlanFixture, `"clauses":[{`, `"clauses":[{"clause_id":"extra","concept_id":"fixture.other","purpose":"candidate_retrieval","relation":"must","forms":[]},{`, 1)
	if result := preflightFromText(t, multi); result.Status != "blocked" {
		t.Fatalf("conjunctive MUST was silently turned into OR: %#v", result)
	}
	tag := strings.Replace(historicalPlanFixture, `"form_type":"alias"`, `"form_type":"source_tag"`, 1)
	if result := preflightFromText(t, tag); result.Status != "blocked" {
		t.Fatalf("unqualified structured tag was lowered to lexical text: %#v", result)
	}
}
