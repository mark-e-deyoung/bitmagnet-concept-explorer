package evidence

import "testing"

func sampleQuery(op string) Selector {
	return Selector{SubjectID: "fictional-1", TargetScope: "person", Role: "attested", PredicateID: "citizenship", ValueID: "A", ValidAt: "2026", Operator: op}
}
func sampleFact(polarity, family string) Fact {
	return Fact{SubjectID: "fictional-1", TargetScope: "person", Role: "attested", PredicateID: "citizenship", ValueID: "A", ValidAt: "2026", Family: family, Polarity: polarity, Qualified: true}
}
func TestEvidenceTruthTable(t *testing.T) {
	cases := []struct {
		name   string
		facts  []Fact
		state  string
		yes    string
		no     string
		strict string
	}{
		{"positive", []Fact{sampleFact("affirmed", "one")}, "supported", "include", "exclude", "exclude"},
		{"negative", []Fact{sampleFact("denied", "one")}, "refuted", "exclude", "include", "include"},
		{"both", []Fact{sampleFact("affirmed", "one"), sampleFact("denied", "two")}, "conflicted", "abstain", "abstain", "abstain"},
		{"neither", nil, "unknown", "abstain", "include_unverified", "abstain"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, op := range []struct{ kind, want string }{{"require", tc.yes}, {"exclude_observed", tc.no}, {"require_absence", tc.strict}} {
				got, err := Evaluate(tc.facts, sampleQuery(op.kind))
				if err != nil {
					t.Fatal(err)
				}
				if got.State != tc.state || got.Outcome != op.want {
					t.Fatalf("%s: %+v; want state=%q outcome=%q", op.kind, got, tc.state, op.want)
				}
			}
		})
	}
}
func TestEvidenceAncestryDedup(t *testing.T) {
	one := sampleFact("affirmed", "mirror-family")
	mirror := one
	no := sampleFact("denied", "independent-family")
	got, err := Evaluate([]Fact{one, mirror, no}, sampleQuery("require"))
	if err != nil {
		t.Fatal(err)
	}
	if got.State != "conflicted" || got.IndependentPositive != 1 || got.IndependentNegative != 1 {
		t.Fatalf("mirror inflated independent sources: %+v", got)
	}
}
func TestNoCrossEntityOrScopeInference(t *testing.T) {
	f := sampleFact("affirmed", "one")
	q := sampleQuery("require")
	f.ValueID = "B" // a different citizenship is NOT proof of A or absence of A
	if got, err := Evaluate([]Fact{f}, q); err != nil || got.State != "unknown" {
		t.Fatalf("different value: %+v %v", got, err)
	}
	f = sampleFact("affirmed", "one")
	f.Role = "parent_birthplace"
	f.PredicateID = "birthplace"
	if got, err := Evaluate([]Fact{f}, q); err != nil || got.State != "unknown" {
		t.Fatalf("parent claimed for subject: %+v %v", got, err)
	}
	f = sampleFact("affirmed", "one")
	f.TargetScope = "work"
	f.Role = "tagged_as"
	f.PredicateID = "activity"
	f.ValueID = "X"
	q = Selector{SubjectID: "fictional-1", TargetScope: "scene", Role: "depicts", PredicateID: "activity", ValueID: "X", ValidAt: "2026", Operator: "require"}
	if got, err := Evaluate([]Fact{f}, q); err != nil || got.State != "unknown" {
		t.Fatalf("work tag became scene claim: %+v %v", got, err)
	}
	f = sampleFact("affirmed", "one")
	f.TargetScope = "image"
	f.Role = "observed"
	f.PredicateID = "hair"
	f.ValueID = "short"
	f.ValidAt = "2024"
	q = Selector{SubjectID: "fictional-1", TargetScope: "image", Role: "observed", PredicateID: "hair", ValueID: "short", ValidAt: "2026", Operator: "require"}
	if got, err := Evaluate([]Fact{f}, q); err != nil || got.State != "unknown" {
		t.Fatalf("past style inferred as current: %+v %v", got, err)
	}
}
func TestUnqualifiedAndInvalidEvidence(t *testing.T) {
	f := sampleFact("affirmed", "one")
	f.Qualified = false
	if got, err := Evaluate([]Fact{f}, sampleQuery("require")); err != nil || got.State != "unknown" {
		t.Fatalf("unqualified accepted: %+v %v", got, err)
	}
	bad := sampleQuery("not")
	if _, err := Evaluate(nil, bad); err == nil {
		t.Fatal("unsupported operator accepted")
	}
	bad = sampleQuery("require")
	bad.SubjectID = ""
	if _, err := Evaluate(nil, bad); err == nil {
		t.Fatal("empty subject accepted")
	}
	f = sampleFact("invalid", "one")
	if _, err := Evaluate([]Fact{f}, sampleQuery("require")); err == nil {
		t.Fatal("bad polarity accepted")
	}
	f = sampleFact("affirmed", "")
	if _, err := Evaluate([]Fact{f}, sampleQuery("require")); err == nil {
		t.Fatal("missing source ancestry accepted")
	}
}
