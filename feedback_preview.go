package main

// A feedback preview binds a researcher-supplied concept gap hypothesis to
// metadata the local Bitmagnet service actually returned in this request.
// It is NOT a CPE adjudicator, evidence authenticator, or submission API.
import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/mark-e-deyoung/bitmagnet-concept-explorer/internal/contracts"
)

type FeedbackPreviewRequest struct {
	Query          Request       `json:"query"`
	Sweep          *SweepRequest `json:"sweep,omitempty"`
	ProposalKind   string        `json:"proposal_kind"`
	CandidateLabel string        `json:"candidate_label"`
	Language       string        `json:"language"`
	ConceptID      string        `json:"concept_id,omitempty"`
	Projection     string        `json:"projection_digest,omitempty"`
}

type FeedbackPreviewResponse struct {
	SchemaVersion    string               `json:"schema_version"`
	State            string               `json:"state"`
	ObservationCount int                  `json:"observation_count"`
	Proposal         *SweepDiscoveryDraft `json:"proposal,omitempty"`
	Warnings         []string             `json:"warnings"`
}

func feedbackHMAC(key []byte, domain string, fields ...string) string {
	m := hmac.New(sha256.New, key)
	_, _ = m.Write([]byte(domain))
	for _, f := range fields {
		_, _ = m.Write([]byte{0})
		_, _ = m.Write([]byte(f))
	}
	return "sha256:" + hex.EncodeToString(m.Sum(nil))
}

func buildFeedbackPreview(ctx context.Context, input FeedbackPreviewRequest, key []byte) (FeedbackPreviewResponse, error) {
	if len(key) < 32 {
		return FeedbackPreviewResponse{}, errors.New("local HMAC key must be at least 32 bytes")
	}
	if input.Query.IncludeWeb {
		return FeedbackPreviewResponse{}, errors.New("feedback preview refuses external web egress")
	}
	// The existing plan validator, not this adapter, owns query and expansion budgets.
	if _, err := makePlan(input.Query); err != nil {
		return FeedbackPreviewResponse{}, err
	}
	if !cpeDiscoveryKinds[input.ProposalKind] || !sweepLanguage(input.Language) ||
		len([]rune(strings.TrimSpace(input.CandidateLabel))) < 2 || len([]rune(input.CandidateLabel)) > 128 {
		return FeedbackPreviewResponse{}, errors.New("invalid proposal candidate")
	}
	if input.Projection != "" && !shaEvidenceRef(input.Projection) {
		return FeedbackPreviewResponse{}, errors.New("invalid pinned projection digest")
	}
	result, err := execute(ctx, input.Query)
	if err != nil {
		return FeedbackPreviewResponse{}, errors.New("local source observation could not execute")
	}
	response := FeedbackPreviewResponse{
		SchemaVersion:    "bce-feedback-preview-v1",
		State:            "non_diagnostic",
		ObservationCount: len(result.Results),
		Warnings: []string{
			"Only bounded current-query Bitmagnet metadata was observed; no full-corpus coverage is implied",
			"Keyed digests protect identifiers against trivial enumeration, not source authenticity or semantic correctness",
			"All unqualified Bitmagnet observations share one source-family ancestry; distinct hashes are not independent semantic corroboration",
		},
	}
	if result.State != "complete" {
		response.State = "source_partial_or_failed_no_proposal"
		return response, nil
	}
	if len(result.Results) == 0 {
		response.State = "no_matching_observation_not_concept_absence"
		return response, nil
	}
	seen := make(map[string]bool)
	ev := make([]SweepDiscoveryEvidence, 0, 16)
	for _, obs := range result.Results {
		if obs.InfoHash == "" || seen[obs.InfoHash] {
			continue
		}
		seen[obs.InfoHash] = true
		ev = append(ev, SweepDiscoveryEvidence{
			Source:      "bitmagnet",
			AncestryRef: feedbackHMAC(key, "bce-bitmagnet-unqualified-source-family-v1", "bitmagnet"),
			EvidenceRef: feedbackHMAC(key, "bce-bitmagnet-observation-v1", obs.InfoHash, obs.Title, input.Query.Query),
		})
		if len(ev) == 16 {
			response.Warnings = append(response.Warnings, "Proposal preview includes at most 16 observations from bounded results")
			break
		}
	}
	if len(ev) == 0 {
		response.State = "unusable_source_identifiers_no_proposal"
		return response, nil
	}
	draft, err := previewSweepDiscovery(SweepDiscoveryRequest{
		Kind: input.ProposalKind, CandidateLabel: input.CandidateLabel,
		Language: input.Language, ConceptID: input.ConceptID,
		SnapshotDigest: input.Projection, Evidence: ev,
	})
	if err != nil {
		return FeedbackPreviewResponse{}, err
	}
	response.Proposal = &draft
	response.State = "local_review_export_only"
	return response, nil
}

func registerFeedbackPreview(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/discovery/feedback/preview", func(w http.ResponseWriter, r *http.Request) {
		key := []byte(os.Getenv("BCE_DISCOVERY_HMAC_KEY"))
		if len(key) < 32 {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error": "local feedback HMAC key not configured", "code": "feedback_not_configured", "state": "not_started",
			})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, contracts.SweepDiscoveryJSONBytes)
		defer r.Body.Close()
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		var input FeedbackPreviewRequest
		if err := dec.Decode(&input); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid feedback request"})
			return
		}
		if err := dec.Decode(new(any)); err != io.EOF {
			writeJSON(w, 400, map[string]string{"error": "trailing feedback request"})
			return
		}
		var out FeedbackPreviewResponse
		var err error
		if input.Sweep != nil {
			if !reflect.DeepEqual(input.Query, Request{}) {
				writeJSON(w, 422, map[string]string{"error": "specify exactly one bounded query or multilingual sweep"})
				return
			}
			out, err = buildFeedbackSweepPreview(r.Context(), input, key)
		} else {
			ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
			defer cancel()
			out, err = buildFeedbackPreview(ctx, input, key)
		}
		if err != nil {
			writeSweepExecutionError(w, err)
			return
		}
		writeJSON(w, 200, out)
	})
}
