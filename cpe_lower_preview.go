package main

import (
	"encoding/json"
	"errors"
	"github.com/mark-e-deyoung/bitmagnet-concept-explorer/internal/contracts"
	"io"
	"net/http"
	"strings"
	"unicode"
)

// Research-only lowering of the historical CPE experimental-v0 tactical IR.
// The accepted CPE projection and policy authority remain elsewhere. Never
// execute this preview, promote model hypotheses, or infer CPE assessment.
type HistoricalBitmagnetQueryPreview struct {
	ClauseID      string         `json:"clause_id"`
	ConceptID     string         `json:"concept_id"`
	FormType      string         `json:"form_type"`
	Language      string         `json:"language"`
	ProvenanceRef string         `json:"provenance_ref"`
	NativeInput   map[string]any `json:"native_input"`
	Status        string         `json:"status"`
}
type HistoricalLowerPreview struct {
	Contract            string                            `json:"contract"`
	Status              string                            `json:"status"`
	ExecutionAuthorized bool                              `json:"execution_authorized"`
	SemanticAuthority   string                            `json:"semantic_authority"`
	Queries             []HistoricalBitmagnetQueryPreview `json:"queries"`
	Warnings            []string                          `json:"warnings"`
}

// Bitmagnet's upstream AppQueryToTsquery lexer treats &, |, !, ., (), * and
// quotes as operators. Until a pinned tokenizer qualification exists,
// allow only letters, marks, numbers, and ASCII single spaces in CPE terms.
// Reject rather than silently distort hyphens, apostrophes or meta syntax.
// Each form is a distinct candidate-retrieval query, not a MUST conjunction.
func bitmagnetSafeLiteral(term string) (string, error) {
	if err := validateBitmagnetSafeLiteral(term); err != nil {
		return "", err
	}
	return `"` + term + `"`, nil
}

func validateBitmagnetSafeLiteral(term string) error {
	if term != strings.TrimSpace(term) {
		return errors.New("lexical term has leading/trailing whitespace or control formatting")
	}
	if term == "" || len([]rune(term)) > 160 {
		return errors.New("lexical term empty or over budget")
	}
	lastSpace := false
	hasLexeme := false
	for _, r := range term {
		if unicode.IsControl(r) {
			return errors.New("lexical term includes controls")
		}
		if r == ' ' {
			if lastSpace {
				return errors.New("lexical term contains repeated whitespace")
			}
			lastSpace = true
			continue
		}
		lastSpace = false
		if !unicode.IsLetter(r) && !unicode.IsMark(r) && !unicode.IsDigit(r) {
			return errors.New("lexical term cannot be lowered without parser ambiguity")
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			hasLexeme = true
		}
	}
	if !hasLexeme {
		return errors.New("lexical term has no searchable lexemes")
	}
	return nil
}
func lowerHistoricalCPEPreview(p HistoricalAdapterPlan) (HistoricalLowerPreview, error) {
	result := HistoricalLowerPreview{
		Contract: "historical-experimental-v0-reference",
		Status:   "blocked", ExecutionAuthorized: false,
		SemanticAuthority: "not_accepted_cpe_projection",
		Queries:           []HistoricalBitmagnetQueryPreview{},
		Warnings: []string{
			"This is a non-executable research preview of a historical, not accepted CPE tactical IR.",
			"Each lexical form is a separate candidate-retrieval query; lexical hits do not prove concept membership.",
			"Bitmagnet language facet is a metadata attribute, not the language of the lexical query.",
			"Native FTS tokenizer/transliteration is not semantically equivalent across languages; qualify results before production.",
		},
	}
	preflight, e := preflightHistoricalCPEPlan(p)
	if e != nil {
		return result, e
	}
	if preflight.Status != "structurally_compatible_not_authorized" {
		return result, errors.New("historical CPE plan failed safety preflight")
	}
	clause := p.TacticalIR.Clauses[0]
	for _, f := range clause.Forms {
		// Never lower source tags or structured mappings into unsafe title FTS.
		switch f.FormType {
		case "preferred_term", "alias", "transliteration", "phrase":
		default:
			return result, errors.New("unsupported historical lexical form type")
		}
		query, e := bitmagnetSafeLiteral(f.Value)
		if e != nil {
			return result, e
		}
		result.Queries = append(result.Queries, HistoricalBitmagnetQueryPreview{
			ClauseID: clause.ClauseID, ConceptID: clause.ConceptID,
			Language: f.Language, FormType: f.FormType, ProvenanceRef: f.ProvenanceRef,
			Status:      "candidate_retrieval_only",
			NativeInput: map[string]any{"queryString": query, "limit": 20, "totalCount": false, "hasNextPage": true},
		})
	}
	if len(result.Queries) == 0 || p.ExecutionConstraints.MaxQueries == nil || len(result.Queries) > *p.ExecutionConstraints.MaxQueries {
		return result, errors.New("historical CPE query budget exceeded")
	}
	result.Status = "lowered_preview_not_authorized"
	return result, nil
}
func registerHistoricalCPELowerPreview(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/cpe/adapter-plan/lower-preview", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, contracts.CPELowerPreviewJSONBytes)
		defer r.Body.Close()
		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields()
		var p HistoricalAdapterPlan
		if e := d.Decode(&p); e != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid historical adapter plan"})
			return
		}
		if e := d.Decode(new(any)); e != io.EOF {
			writeJSON(w, 400, map[string]string{"error": "trailing historical adapter plan"})
			return
		}
		out, e := lowerHistoricalCPEPreview(p)
		if e != nil {
			writeJSON(w, 422, map[string]string{"error": e.Error()})
			return
		}
		writeJSON(w, 200, out)
	})
}
