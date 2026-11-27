package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mark-e-deyoung/bitmagnet-concept-explorer/internal/contracts"
)

// ComparisonRecord contains caller-reported metadata only, not authenticated
// provider evidence. No torrent hashes, resource locators or acquisition fields.
type ComparisonRecord struct {
	Title   string            `json:"title,omitempty"`
	Kind    string            `json:"kind,omitempty"`
	Year    int               `json:"year,omitempty"`
	Edition string            `json:"edition,omitempty"`
	Creator string            `json:"creator,omitempty"`
	Format  string            `json:"format,omitempty"`
	IDs     map[string]string `json:"catalog_ids,omitempty"`
}

type ComparisonObservation struct {
	Source string           `json:"source"`
	State  string           `json:"state"`
	Record ComparisonRecord `json:"record"`
}

type MetadataComparisonRequest struct {
	Reference    ComparisonRecord        `json:"reference"`
	Observations []ComparisonObservation `json:"observations"`
}

type ComparisonOutcome struct {
	Index       int               `json:"index"`
	Source      string            `json:"source"`
	SourceState string            `json:"source_state"`
	Identity    string            `json:"identity"`
	Fields      map[string]string `json:"fields"`
	Ancestry    string            `json:"ancestry"`
}

type MetadataComparisonResponse struct {
	Version              string              `json:"version"`
	ReadOnly             bool                `json:"read_only"`
	ExternalRequests     int                 `json:"external_requests"`
	IndependentWitnesses int                 `json:"independent_witnesses"`
	Outcomes             []ComparisonOutcome `json:"outcomes"`
	Warnings             []string            `json:"warnings"`
}

var compareIDPatterns = map[string]*regexp.Regexp{
	"imdb":        regexp.MustCompile(`^tt[0-9]{7,10}$`),
	"tmdb_movie":  regexp.MustCompile(`^[1-9][0-9]{0,10}$`),
	"tmdb_series": regexp.MustCompile(`^[1-9][0-9]{0,10}$`),
	"isbn13":      regexp.MustCompile(`^[0-9]{13}$`),
	"musicbrainz": regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`),
}

func comparisonTextOK(s string) bool {
	if !utf8.ValidString(s) || utf8.RuneCountInString(s) > 256 {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func validateComparisonRecord(r ComparisonRecord) error {
	for _, s := range []string{r.Title, r.Kind, r.Edition, r.Creator, r.Format} {
		if !comparisonTextOK(s) {
			return errors.New("invalid comparison text")
		}
	}
	if r.Year != 0 && (r.Year < 1 || r.Year > 2100) {
		return errors.New("invalid comparison year")
	}
	if len(r.IDs) > 5 {
		return errors.New("catalog ID budget exceeded")
	}
	for ns, id := range r.IDs {
		pattern, ok := compareIDPatterns[ns]
		if !ok || len(id) > 64 || !pattern.MatchString(id) {
			return errors.New("invalid catalog ID claim")
		}
		if ns == "tmdb_movie" && r.Kind != "" && r.Kind != "movie" {
			return errors.New("catalog ID kind mismatch")
		}
		if ns == "tmdb_series" && r.Kind != "" && r.Kind != "series" {
			return errors.New("catalog ID kind mismatch")
		}
	}
	return nil
}

func comparisonField(a, b string) string {
	a, b = strings.Join(strings.Fields(a), " "), strings.Join(strings.Fields(b), " ")
	if a == "" || b == "" {
		return "unknown"
	}
	if strings.EqualFold(a, b) {
		return "agreement"
	}
	return "difference"
}

func compareMetadata(in MetadataComparisonRequest) (MetadataComparisonResponse, error) {
	if len(in.Observations) == 0 || len(in.Observations) > 16 {
		return MetadataComparisonResponse{}, errors.New("1-16 observations required")
	}
	if err := validateComparisonRecord(in.Reference); err != nil {
		return MetadataComparisonResponse{}, err
	}
	if strings.TrimSpace(in.Reference.Title) == "" && len(in.Reference.IDs) == 0 {
		return MetadataComparisonResponse{}, errors.New("reference title or catalog ID required")
	}
	out := MetadataComparisonResponse{
		Version:  "metadata-comparison-preview-v1",
		ReadOnly: true,
		Outcomes: []ComparisonOutcome{},
		Warnings: []string{
			"Caller-reported observations are not authenticated sources; shared identifiers are claims, not verified identities",
			"Titles and field agreements do not establish torrent identity, authorization, safety or CPE semantic acceptance",
			"Unresolved source ancestry is conservatively shared; no independent corroboration or absence is asserted",
			"Only comparison states are returned; no titles, identifiers, links or payloads are returned or persisted",
		},
	}
	for i, obs := range in.Observations {
		switch obs.Source {
		case "bitmagnet", "searxng", "reference_catalog", "publisher", "jellyfin", "tmdb", "imdb":
		default:
			return MetadataComparisonResponse{}, errors.New("unsupported comparison source role")
		}
		switch obs.State {
		case "observed", "partial", "unavailable", "refused", "no_match":
		default:
			return MetadataComparisonResponse{}, errors.New("unsupported comparison source state")
		}
		if err := validateComparisonRecord(obs.Record); err != nil {
			return MetadataComparisonResponse{}, err
		}
		row := ComparisonOutcome{
			Index:       i,
			Source:      obs.Source,
			SourceState: obs.State,
			Identity:    "unknown",
			Fields:      map[string]string{},
			Ancestry:    "shared_unqualified",
		}
		if obs.State != "observed" {
			row.Identity = "source_incomplete_non_diagnostic"
			out.Outcomes = append(out.Outcomes, row)
			continue
		}
		a, b := in.Reference, obs.Record
		row.Fields["title"] = comparisonField(a.Title, b.Title)
		row.Fields["kind"] = comparisonField(a.Kind, b.Kind)
		row.Fields["edition"] = comparisonField(a.Edition, b.Edition)
		row.Fields["creator"] = comparisonField(a.Creator, b.Creator)
		row.Fields["format"] = comparisonField(a.Format, b.Format)
		row.Fields["year"] = "unknown"
		if a.Year != 0 && b.Year != 0 {
			row.Fields["year"] = "difference"
			if a.Year == b.Year {
				row.Fields["year"] = "agreement"
			}
		}
		idAgree, idConflict := false, false
		for ns, left := range a.IDs {
			state := comparisonField(left, b.IDs[ns])
			row.Fields["catalog_id:"+ns] = state
			idAgree = idAgree || state == "agreement"
			idConflict = idConflict || state == "difference"
		}
		if idConflict || row.Fields["kind"] == "difference" || row.Fields["year"] == "difference" || row.Fields["edition"] == "difference" {
			row.Identity = "conflicting_claims_review_required"
		} else if idAgree {
			row.Identity = "shared_catalog_id_claim_unverified"
		} else if row.Fields["title"] == "agreement" {
			row.Identity = "title_candidate_unconfirmed"
		}
		out.Outcomes = append(out.Outcomes, row)
	}
	return out, nil
}

func registerMetadataComparison(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/metadata/compare", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, contracts.MetadataCompareJSONBytes)
		defer r.Body.Close()
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		var in MetadataComparisonRequest
		if err := dec.Decode(&in); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid metadata comparison request"})
			return
		}
		if err := dec.Decode(new(any)); err != io.EOF {
			writeJSON(w, 400, map[string]string{"error": "trailing or oversized metadata comparison request"})
			return
		}
		out, err := compareMetadata(in)
		if err != nil {
			writeJSON(w, 422, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, out)
	})
}
