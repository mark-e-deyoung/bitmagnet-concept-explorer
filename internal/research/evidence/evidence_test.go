package evidence

import "testing"

func baseQuery(op string) Selector {
	return Selector{
		SubjectID: "fictional-person-01", TargetScope: "person",
		Role: "source_attested", PredicateID: "citizenship", ValueID: "country-a",
		ValidAt: "2026-01-01", Operator: op,
	}
}

func baseFact(polarity string) Fact {
	return Fact{
		SubjectID: "fictional-person-01", TargetScope: "person",
		Role: "source_attested", PredicateID: "citizenship", ValueID: "country-a",
		ValidAt: "2026-01-01", Family: "source-family-1",
		Polarity: polarity, Qualified: true,
	}
}

func TestFourEvidenceStatesAndTypedNegation(t *testing.T) {
	yes := baseFact("affirmed")
	no := baseFact("denied")
	cases := []struct {
		name string
		facts []Fact
		state string
		require string
		exclude string
		absence string
	}{
		{"positive", []Fact{yes}, "supported", "include", "exclude", "exclude"},
		{"negative", []Fact{no}, "refuted", "exclude", "include", "include"},
		{"contradiction", []Fact{yes, no}, "conflicted", "abstain", "abstain", "abstain"},
		{"no-evidence", nil, "unknown", "abstain", "include_unverified", "abstain"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, op := range []struct{ kind, want string }{
				{"require", tc.require},
				{"exclude_observed", tc.exclude},
				{"require_absence", tc.absence},
			} {
				got, err := Evaluate(tc.facts, baseQuery(op.kind))
				if err != nil { t.Fatal(err) }
				if got.State != tc.state || got.Outcome != op.want {
					t.Fatalf("%s: state/outcome = %q/%q, want %q/%q", op.kind, got.State, got.Outcome, tc.state, op.want)
				}
			}
		})
	}
}

func TestMirrorAncestryAndIndependentContradiction(t *testing.T) {
	one := baseFact("affirmed")
	mirror := one // a different indexer copying exactly the same source family
	other := baseFact("denied")
	other.Family = "independent-source"
	got, err := Evaluate([]Fact{one, mirror, other}, baseQuery("require"))
	if err != nil { t.Fatal(err) }
	if got.State != "conflicted" || got.IndependentPositive != 1 || got.IndependentNegative != 1 {
		t.Fatalf("mirrors inflated evidence or conflict was lost: %+v", got)
	}
}

func TestNoCrossPersonParentRoleWorkOrTimeInference(t *testing.T) {
	yes := baseFact("affirmed")
	differentCitizenship := yes
	differentCitizenship.ValueID = "country-b"
	parentBirthplace := yes
	parentBirthplace.Role = "parent_birthplace"
	parentBirthplace.PredicateID = "birthplace"
	workTag := yes
	workTag.TargetScope = "work"
	workTag.Role = "tagged_as"
	workTag.PredicateID = "activity"
	workTag.ValueID = "fictional-action"
	oldImage := yes
	oldImage.TargetScope = "image"
	oldImage.Role = "image_observed"
	oldImage.PredicateID = "hair_style"
	oldImage.ValueID = "short"
	oldImage.ValidAt = "2024-01-01"
	for _, x := range []struct{name string; facts []Fact; q Selector}{
		{"other-citizenship-is-not-absence", []Fact{differentCitizenship}, baseQuery("require_absence")},
		{"parent-is-not-subject-citizenship", []Fact{parentBirthplace}, baseQuery("require")},
		{"tagged-work-is-not-scene-depiction", []Fact{workTag}, Selector{SubjectID: yes.SubjectID,TargetScope:"scene",Role:"depicts",PredicateID:"activity",ValueID:"fictional-action",ValidAt:yes.ValidAt,Operator:"require"}},
		{"image-at-other-time-is-not-current", []Fact{oldImage}, Selector{SubjectID: yes.SubjectID,TargetScope:"image",Role:"image_observed",PredicateID:"hair_style",ValueID:"short",ValidAt:"2026-01-01",Operator:"require"}},
	} {
		t.Run(x.name, func(t *testing.T) {
			got, err := Evaluate(x.facts, x.q)
			if err != nil { t.Fatal(err) }
			if got.State != "unknown" || got.Outcome != "abstain" {
				t.Fatalf("unsupported inference from another relation: %+v", got)
			}
		})
	}
}

func TestUnqualifiedFactsAndInvalidInputs(t *testing.T) {
	unqualified := baseFact("affirmed")
	unqualified.Qualified = false
	got, err := Evaluate([]Fact{unqualified}, baseQuery("require"))
	if err != nil || got.State != "unknown" { t.Fatalf("unqualified data gained authority: %+v, %v", got, err) }

	q := baseQuery("not")
	if _, err := Evaluate(nil, q); err == nil { t.Fatal("unrecognized operator accepted") }
	q = baseQuery("require")
	q.SubjectID = ""
	if _, err := Evaluate(nil, q); err == nil { t.Fatal("missing subject accepted") }
	bad := baseFact("uncertain")
	if _, err := Evaluate([]Fact{bad}, baseQuery("require")); err == nil {
		t.Fatal("unrecognized qualified polarity accepted")
	}
	bad = baseFact("affirmed")
	bad.Family = ""
	if _, err := Evaluate([]Fact{bad}, baseQuery("require")); err == nil {
		t.Fatal("qualified evidence without ancestry accepted")
	}
}
