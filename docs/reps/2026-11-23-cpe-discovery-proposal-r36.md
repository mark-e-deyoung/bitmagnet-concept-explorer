# R36 — Bounded ConceptDiscoveryProposal from independent research consumer

**2026-11-23 | DRAFT | PUBLIC-SAFE SYNTHETIC ONLY | NO SEMANTIC AUTHORITY**

Governing CPE DLE: [#202](https://github.com/mark-e-deyoung/concept-policy-engine-private/issues/202) discovery flywheel; #201 Concept Authority, #348 federation, #79 independent assurance. Parent [BCE R35 #51](https://github.com/mark-e-deyoung/bitmagnet-concept-explorer/pull/51) exact `3e66404fd0188afd4c809441827f9fe3ea585cb2`. This is a **proposal research envelope**, not an accepted, version-frozen common schema or a replacement for any source-specific search adapter, Bitmagnet metadata owner, or CPE reconciler.

## User flow and bounded contract

The independently implemented Python R35 read consumer validates `source_lock_sha256` and exact snapshot **byte** SHA-256 against **externally supplied expected values**, reads an immutable `research_only` projection and observes one *synthetic* consumer/source event. New `discovery_r36.py` produces a deterministic `cpe-discovery-proposal-research-v0` envelope with:

- ID = SHA256 of a domain-separated canonicalized UTF-8 JSON proposal body (the ID is research-only, not CPE semantic identity);
- fixed `RESEARCH_PROPOSAL_ONLY` authority and `UNADJUDICATED` decision state;
- producer name and **full exact 40-character Git revision**; source lock **and** exact snapshot byte pin;
- target existing or unknown stable ID; BCP47-like source language; optional bounded native source term;
- provenance input witness digest and ancestry; `support_count=1`, `independent_corroboration=NOT_ESTABLISHED` and no invented confidence;
- privacy statement `synthetic_public_only`, source-system ID, and no raw source URLs, filesystem paths or private corpus payload.

**Decision rule is intentionally conservative:**

- Already-present exact same-language term → `ALREADY_OBSERVED_NO_PROPOSAL`.
- New native term for *known* target concept → `lexicalization_candidate`, equivalence `UNKNOWN`, unadjudicated.
- Unknown target concept → `coverage_gap`, **NOT** an automatic new concept.
- Existing concept but failed retrieval → `coverage_gap` attributed to capability/projection uncertainty, **NOT** ontology mutation.

All cases remain *candidate research* even when source evidence repeats or when similar terms co-occur. No semantic promotion, relation/split/merge, multilingual exact-equivalence claim, policy disposition, source rights, acceptance or privacy declassification is performed.

## Offline usage (synthetic only)

```sh
go run ./cmd/cpepack -packages testdata/cpepack/packages.json \
  -lock testdata/cpepack/lock.json -root test.public.literature > /tmp/cpe-research.json

python3 -B cmd/cpepack/discovery_r36.py \
  --snapshot /tmp/cpe-research.json \
  --expected-lock-sha256 <separately-admitted-64-hex> \
  --expected-snapshot-sha256 <separately-admitted-exact-byte-64-hex> \
  --concept-id public:literary-work --language th \
  --observation-type native_source_term --term 'งานวรรณกรรม' \
  --source-system fixture.public.lexical.source \
  --evidence-sha256 <synthetic-source-witness-64-hex> \
  --producer-revision 3e66404fd0188afd4c809441827f9fe3ea585cb2
```

The source/byte pin can only be called *authorized* if supplied by the independent owner/issuer, not computed from the untrusted candidate in production. The CI demonstration computes expectations from the original synthetic fixture solely to prove transport mechanics and byte invariants.

## Qualification / differential goals

Native Python stdlib tests cover deterministic repeat, same-source non-independent support, exact known-term dedup, missing lex vs explicitly no-equivalent, unknown-concept non-novelty, consumer-vs-semantic gap, invalid producer revision/source witness, invalid language/source system, malformed Unicode, oversized/private terms, and byte-tamper refusal. The original public synthetic Go workflow is extended to run these tests and a **real Go-generated projection → independent Python R35 reader → R36 lexical candidate** CLI rep.

Audience contracts: human-readable CLI help; automation stable typed JSON, no network, exit 3 on refused input; agents can route `proposal_kind`, `mapping.equivalence_status=UNKNOWN`, provenance ancestry, and `UNADJUDICATED` to human/independent CPE research intake. **No direct write into accepted CPE.**

## Known REDs / holds

One producer with one fixture source does **not** satisfy #202's three consumer *classes*, genuine independent multi-source corroboration, corpus privacy review, ontology-vs-capability adjudication, offline/online retrieval metrics, signed immutable accepted CA snapshot, or promotion/replay. Even a successful native CI has no declassification or privacy authority: only original synthetic public terms allowed. Future separate reps must validate candidate against independent native-language source and counterexample; source ancestry must be deduplicated before scoring corroboration. Keep S4/S5, real rights/MAF custody, #79 freeze, and merge/release/deploy gates held.
