// Package evidence is an offline, research-only model of qualified source claims.
// It is intentionally not imported by the running Explorer or by CPE authority.
package evidence

import (
	"errors"
	"strings"
)

// Fact identifies one source-scoped and time-scoped assertion.
// Role distinguishes e.g. credited_in from depicts; source copies share Family.
type Fact struct {
	SubjectID   string
	TargetScope string
	Role        string
	PredicateID string
	ValueID     string
	ValidAt     string
	Family      string
	Polarity    string // affirmed or denied; not a source keyword NOT
	Qualified   bool
}

// Selector is a typed query over evidence, not Bitmagnet FTS syntax.
type Selector struct {
	SubjectID   string
	TargetScope string
	Role        string
	PredicateID string
	ValueID     string
	ValidAt     string
	Operator    string // require, exclude_observed, require_absence
}

type Evaluation struct {
	State               string // supported, refuted, conflicted, unknown
	Outcome             string // include, exclude, include_unverified, abstain
	IndependentPositive int
	IndependentNegative int
}

// Evaluate keeps subject, role, resource scope and time exact. It deliberately
// does not infer facts from relatives, titles, credited roles or scene context.
func Evaluate(facts []Fact, q Selector) (Evaluation, error) {
	if strings.TrimSpace(q.SubjectID) == "" || strings.TrimSpace(q.TargetScope) == "" ||
		strings.TrimSpace(q.Role) == "" || strings.TrimSpace(q.PredicateID) == "" ||
		strings.TrimSpace(q.ValueID) == "" {
		return Evaluation{}, errors.New("query missing typed identity or scope")
	}
	switch q.Operator {
	case "require", "exclude_observed", "require_absence":
	default:
		return Evaluation{}, errors.New("unsupported typed operator")
	}
	pos, neg := map[string]bool{}, map[string]bool{}
	for _, f := range facts {
		if !f.Qualified {
			continue
		}
		if f.Family == "" || (f.Polarity != "affirmed" && f.Polarity != "denied") {
			return Evaluation{}, errors.New("qualified fact has invalid source ancestry or polarity")
		}
		if f.SubjectID != q.SubjectID || f.TargetScope != q.TargetScope ||
			f.Role != q.Role || f.PredicateID != q.PredicateID ||
			f.ValueID != q.ValueID || f.ValidAt != q.ValidAt {
			continue
		}
		if f.Polarity == "affirmed" {
			pos[f.Family] = true
		} else {
			neg[f.Family] = true
		}
	}
	r := Evaluation{IndependentPositive: len(pos), IndependentNegative: len(neg)}
	switch {
	case len(pos) > 0 && len(neg) > 0:
		r.State = "conflicted"
	case len(pos) > 0:
		r.State = "supported"
	case len(neg) > 0:
		r.State = "refuted"
	default:
		r.State = "unknown"
	}
	switch q.Operator {
	case "require":
		switch r.State {
		case "supported":
			r.Outcome = "include"
		case "refuted":
			r.Outcome = "exclude"
		default:
			r.Outcome = "abstain"
		}
	case "exclude_observed":
		switch r.State {
		case "supported":
			r.Outcome = "exclude"
		case "refuted":
			r.Outcome = "include"
		case "unknown":
			r.Outcome = "include_unverified"
		default:
			r.Outcome = "abstain"
		}
	case "require_absence":
		switch r.State {
		case "refuted":
			r.Outcome = "include"
		case "supported":
			r.Outcome = "exclude"
		default:
			r.Outcome = "abstain"
		}
	}
	return r, nil
}
