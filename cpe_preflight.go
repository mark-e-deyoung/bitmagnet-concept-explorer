package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// These are HISTORICAL experimental-v0 CPE contract shapes, not adopted
// Concept Authority or production search-plan schemas. Parsing is read-only.
type HistoricalAdapterPlan struct {
	SchemaVersion              string   `json:"schema_version"`
	ExecutionID                string   `json:"execution_id"`
	SearchPlanRef              string   `json:"search_plan_ref"`
	TargetCapabilityProfileRef string   `json:"target_capability_profile_ref"`
	Status                     string   `json:"status"`
	RequiredTargetCapabilities []string `json:"required_target_capabilities"`
	OptionalTargetCapabilities []string `json:"optional_target_capabilities"`
	TacticalIR                 struct {
		Clauses []struct {
			ClauseID  string `json:"clause_id"`
			ConceptID string `json:"concept_id"`
			Purpose   string `json:"purpose"`
			Relation  string `json:"relation"`
			Forms     []struct {
				Value         string `json:"value"`
				Language      string `json:"language"`
				FormType      string `json:"form_type"`
				ProvenanceRef string `json:"provenance_ref"`
				CollisionRisk string `json:"collision_risk,omitempty"`
			} `json:"forms"`
		} `json:"clauses"`
	} `json:"tactical_ir"`
	ExecutionConstraints struct {
		MutationAllowed               bool `json:"mutation_allowed"`
		MaxQueries                    *int `json:"max_queries"`
		AuthenticatedExecutionAllowed bool `json:"authenticated_execution_allowed"`
	} `json:"execution_constraints"`
	PrivacyEffect struct {
		ExternalDisclosure     string   `json:"external_disclosure"`
		Authentication         string   `json:"authentication"`
		CrossSurfaceIdentifier bool     `json:"cross_surface_identifier"`
		LocalQueryHistory      string   `json:"local_query_history"`
		TargetRetention        string   `json:"target_retention"`
		CorrelationRisk        string   `json:"correlation_risk"`
		ResidualRisks          []string `json:"residual_risks"`
	} `json:"privacy_effect"`
	ObservationRequirements struct {
		EmitNormalizedObservations bool `json:"emit_normalized_observations"`
		PreserveTargetCoverage     bool `json:"preserve_target_coverage"`
		NoMatchIsNegativeEvidence  bool `json:"no_match_is_negative_evidence"`
	} `json:"observation_requirements"`
}
type CPEPreflightResult struct {
	Status                 string   `json:"status"`
	Contract               string   `json:"contract"`
	ExecutionAuthorized    bool     `json:"execution_authorized"`
	SemanticAuthority      string   `json:"semantic_authority"`
	CompatibleLexicalForms int      `json:"compatible_lexical_forms"`
	Reasons                []string `json:"reasons"`
}

func preflightHistoricalCPEPlan(p HistoricalAdapterPlan) (CPEPreflightResult, error) {
	out := CPEPreflightResult{Status: "blocked", Contract: "historical-experimental-v0-reference", ExecutionAuthorized: false, SemanticAuthority: "not_accepted_cpe_projection", Reasons: []string{}}
	if p.SchemaVersion != "experimental-v0" || p.ExecutionID == "" || p.SearchPlanRef == "" || p.TargetCapabilityProfileRef == "" {
		return out, errors.New("invalid historical CPE adapter plan identity")
	}
	if p.Status != "ready" {
		out.Reasons = append(out.Reasons, "plan not ready without unimplemented postchecks or owner authorization")
	}
	if p.ExecutionConstraints.MutationAllowed || p.ExecutionConstraints.AuthenticatedExecutionAllowed {
		out.Reasons = append(out.Reasons, "mutation or authenticated operation requested")
	}
	if p.PrivacyEffect.CrossSurfaceIdentifier || p.PrivacyEffect.LocalQueryHistory == "durable" || p.PrivacyEffect.Authentication != "none" {
		out.Reasons = append(out.Reasons, "privacy boundary cannot be preserved")
	}
	if p.PrivacyEffect.ExternalDisclosure != "none" {
		out.Reasons = append(out.Reasons, "external disclosure requires distinct explicit source authorization")
	}
	if !p.ObservationRequirements.EmitNormalizedObservations || !p.ObservationRequirements.PreserveTargetCoverage || p.ObservationRequirements.NoMatchIsNegativeEvidence {
		out.Reasons = append(out.Reasons, "required observation or unknown semantics violated")
	}
	if len(p.TacticalIR.Clauses) != 1 {
		out.Reasons = append(out.Reasons, "current Go executor cannot lower multi-clause conjunction without semantic loss")
	}
	if len(p.TacticalIR.Clauses) == 1 {
		c := p.TacticalIR.Clauses[0]
		if c.ClauseID == "" || c.ConceptID == "" || c.Relation != "must" || c.Purpose != "candidate_retrieval" {
			out.Reasons = append(out.Reasons, "unsupported clause purpose or relation")
		}
		if len(c.Forms) == 0 || len(c.Forms) > 32 {
			out.Reasons = append(out.Reasons, "invalid lexical fan-out bound")
		}
		for _, f := range c.Forms {
			if f.Value == "" || len([]rune(f.Value)) > 512 || f.Language == "" || len(f.Language) > 32 || f.ProvenanceRef == "" {
				out.Reasons = append(out.Reasons, "invalid source-linked lexical form")
				continue
			}
			switch f.FormType {
			case "preferred_term", "alias", "transliteration", "phrase":
				out.CompatibleLexicalForms++
			default:
				out.Reasons = append(out.Reasons, "source-tag/descriptor form not supported by lexical search alone")
			}
		}
		if p.ExecutionConstraints.MaxQueries == nil || *p.ExecutionConstraints.MaxQueries < len(c.Forms) || *p.ExecutionConstraints.MaxQueries > 32 {
			out.Reasons = append(out.Reasons, "query budget insufficient or unbounded")
		}
	}
	for _, cap := range p.RequiredTargetCapabilities {
		switch cap {
		case "keyword_search", "unicode_query", "json_results", "language_selection":
		default:
			out.Reasons = append(out.Reasons, "target required capability not qualified: "+cap)
		}
	}
	if len(p.TargetCapabilityProfileRef) > 256 || len(p.TacticalIR.Clauses) > 32 || len(p.RequiredTargetCapabilities) > 32 {
		return out, errors.New("historical CPE plan exceeds bounds")
	}
	// A clean *structural* preflight is still not proof of accepted semantic CPE authority.
	if len(out.Reasons) == 0 {
		out.Status = "structurally_compatible_not_authorized"
	}
	return out, nil
}
func registerHistoricalCPEPreflight(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/cpe/adapter-plan/preflight", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 32768)
		defer r.Body.Close()
		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields()
		var p HistoricalAdapterPlan
		if e := d.Decode(&p); e != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid or policy-leaking historical adapter-plan JSON"})
			return
		}
		if e := d.Decode(new(any)); e != io.EOF {
			writeJSON(w, 400, map[string]string{"error": "trailing historical CPE adapter-plan JSON"})
			return
		}
		out, e := preflightHistoricalCPEPlan(p)
		if e != nil {
			writeJSON(w, 422, map[string]string{"error": e.Error()})
			return
		}
		// Do not return raw lexical forms, privacy fields or policy state.
		writeJSON(w, 200, out)
	})
}
