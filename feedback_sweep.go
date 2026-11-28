package main

// Explicit multilingual sweep feedback uses the EXISTING bounded local
// Bitmagnet and optional consent-gated private SearXNG engines. Only keyed
// evidence digests leave this preview; the legacy sweep's unkeyed SHA-256
// references are deliberately not re-used for a CPE research proposal.
import (
	"context"
	"errors"
	"fmt"
	"strings"
)

func buildFeedbackSweepPreview(ctx context.Context, input FeedbackPreviewRequest, key []byte) (FeedbackPreviewResponse, error) {
	if len(key) < 32 || input.Sweep == nil {
		return FeedbackPreviewResponse{}, errors.New("sweep feedback requires a local keyed evidence configuration and sweep request")
	}
	if !cpeDiscoveryKinds[input.ProposalKind] || !sweepLanguage(input.Language) ||
		len([]rune(strings.TrimSpace(input.CandidateLabel))) < 2 ||
		len([]rune(input.CandidateLabel)) > 128 {
		return FeedbackPreviewResponse{}, errors.New("invalid sweep feedback hypothesis")
	}
	if input.Projection != "" && !shaEvidenceRef(input.Projection) {
		return FeedbackPreviewResponse{}, errors.New("invalid pinned research projection digest")
	}
	// Reject unsupported external plans before making any provider requests.
	plan, err := makeSweepPlan(*input.Sweep)
	if err != nil {
		return FeedbackPreviewResponse{}, err
	}
	if input.Projection != "" && input.Projection != plan.ProjectionDigest {
		return FeedbackPreviewResponse{}, errors.New("sweep projection digest differs from selected source")
	}
	result, err := runSweep(ctx, *input.Sweep)
	if err != nil {
		return FeedbackPreviewResponse{}, fmt.Errorf("bounded sweep did not execute: %w", err)
	}
	out := FeedbackPreviewResponse{
		SchemaVersion: "bce-feedback-preview-v1",
		State:         "non_diagnostic",
		Warnings: []string{
			"Bounded multilingual retrieval observations are not accepted CPE concept evidence",
			"Public search engine suggestions and Bitmagnet hits are not independent semantic corroboration",
			"Every unresolved source is assigned ONE conservative, unqualified ancestry across Bitmagnet and SearXNG",
			"Keyed references protect against trivial dictionary correlation, not semantic authenticity, rights or issuer identity",
			"Local preview only: no automatic CPE intake, storage, source promotion or external sharing",
		},
	}
	if result.State != "complete" || result.CPEWrite {
		out.State = "source_partial_or_failed_no_proposal"
		return out, nil
	}
	seen := map[string]bool{}
	evidence := make([]SweepDiscoveryEvidence, 0, 16)
	ancestry := feedbackHMAC(key, "bce-sweep-unqualified-connected-source-family-v1", "all")
	appendRef := func(source, ref string) {
		if ref == "" || seen[ref] || len(evidence) == 16 {
			return
		}
		seen[ref] = true
		evidence = append(evidence, SweepDiscoveryEvidence{Source: source,
			EvidenceRef: ref, AncestryRef: ancestry})
	}
	if result.Bitmagnet != nil {
		for _, item := range result.Bitmagnet.Results {
			if item.InfoHash != "" {
				appendRef("bitmagnet", feedbackHMAC(key, "bce-sweep-local-observation-v1", item.InfoHash, item.Title))
			}
		}
	}
	for _, observed := range result.External {
		for _, hit := range observed.Hits {
			if hit.URL != "" && hit.Title != "" {
				appendRef("searxng", feedbackHMAC(key, "bce-sweep-external-observation-v1",
					hit.URL, hit.Title, observed.Language))
			}
		}
		for _, hint := range observed.Suggestions {
			if strings.TrimSpace(hint) != "" {
				appendRef("searxng", feedbackHMAC(key, "bce-sweep-source-suggestion-v1",
					observed.Term, hint, observed.Language))
			}
		}
	}
	out.ObservationCount = len(evidence)
	if len(evidence) == 0 {
		out.State = "no_matching_observation_not_concept_absence"
		return out, nil
	}
	if len(evidence) == 16 {
		out.Warnings = append(out.Warnings, "Feedback preview includes at most 16 keyed observations from bounded first pages")
	}
	projection := input.Projection
	if projection == "" {
		projection = plan.ProjectionDigest
	}
	draft, err := previewSweepDiscovery(SweepDiscoveryRequest{
		Kind: input.ProposalKind, CandidateLabel: input.CandidateLabel,
		Language: input.Language, ConceptID: input.ConceptID,
		SnapshotDigest: projection, Evidence: evidence,
	})
	if err != nil {
		return FeedbackPreviewResponse{}, err
	}
	out.Proposal = &draft
	out.State = "local_review_export_only"
	return out, nil
}
