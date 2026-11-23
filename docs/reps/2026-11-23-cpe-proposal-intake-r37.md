# R37 — Independent offline Go research intake of Python ConceptDiscoveryProposal

**2026-11-23 · DRAFT / SYNTHETIC_PUBLIC_ONLY / NOT A REAL AUTHORITY RECONCILER OR PROMOTION SERVICE**

Owner CPE #202 governs consumer proposal interpretation; #201 semantic authority, #348 federation, #79 independent acceptance. Public BCE research PR chain: #51 Go-produced readmodel/Python R35 independent consumer → #52 R36 minimal proposal → **R37** separate Go stdlib-only read-verifier. Exact parent [BCE #52](https://github.com/mark-e-deyoung/bitmagnet-concept-explorer/pull/52) at `2e70eeee523ca723401b0c65dfae6ab311509761`.

## Why independent intake is necessary

A proposal is untrusted input, even when it carries an apparently correct `proposal_id`, `source_lock_sha256`, or `RESEARCH_PROPOSAL_ONLY` string. The producer must not self-accept research as canonical meaning. A CPE-side verifier needs independently admitted expectations and a fail-closed typed boundary **before** human/independent semantic adjudication. R37 demonstrates that mechanism with a minimal independent Go CLI, not a central service, database, queue, or replacement for actual CPE #276 reconciliation.

The CLI `cmd/cpeproposal` independently reads R36's output and rejects:
- overlarge (>64 KiB), malformed/duplicate-key/invalid UTF-8 proposal bytes, including nested Unicode-escaped duplicate keys; unrecognized top-level and nested fields;
- invalid version, `authority` or non-`UNADJUDICATED` status; unauthorized mapping `EXACT` or `candidate_not_accepted=false`;
- absent or mismatched **externally expected** source lock, exact snapshot byte SHA, exact 40-character producer Git revision, source-system ID and one original source evidence digest;
- fabricated independent corroboration or `support_count>1` from a single ancestry item;
- fake independent private sanitization/rights authority;
- stale/rewritten proposal body whose research `proposal_id` no longer matches the canonical body digest;
- malformed lexicalization/coverage-gap classification inconsistent with consumer observation type.

It outputs only `VALID_RESEARCH_PROPOSAL_ONLY`, `DEFER_INDEPENDENT_GROUNDING`, `RESEARCH_ONLY`, `UNKNOWN`, `NOT_ESTABLISHED`, and `accepted_semantic_changes=0`. No accepted concept/fact/policy data is written, queried or promoted. No HTTP/network or external dependencies; file-scoped execution ends after one JSON response.

## Synthetic native integration

The public `cpe-lock-source-qualification.yml` runs the existing Go source/race/vet/crossbuild tests plus R35/R36 Python tests, then runs `go test -count=2 ./cmd/cpeproposal`, `go test -race`, `go vet`, and Linux amd64/arm64 builds. End-to-end:

```text
Go cpepack.Build -> original public-safe research JSON
 -> independent Python R35 exact-source/byte-pinned consumer
 -> R36 synthetic Thai lexicalization candidate
 -> independent Go R37 typed/intact proposal validator
 -> DEFER_INDEPENDENT_GROUNDING, 0 accepted changes
```

Expected source/byte and source ancestry digests in CI are deliberately synthetic and derived from the fixture/synthetic event. This proves a cross-language interchange and field/digest checks, **not a signed trusted external issuer, source rights, genuine provenance, a production privacy review, or a passed CPE promotion gate**. The Go proposal ID comparison is *self-consistency* under a versioned research canonical-JSON algorithm; absent a trusted upstream pin/signature, it is NOT authenticity.

## Explicit technical debts / next independent RED

- The Go validator is a source-safe research prototype, not official #202 ingestion or persistent dedup. A second proposal from the same source ancestry must not be miscounted as independent; a real intake must consolidate historical proposals and conflict/withdrawal records.
- R36 privacy is self-labeled `synthetic_public_only`, and actual private terms cannot be copied into public workflows; proper privacy review, minimization and source-rights qualification remain external.
- JSON encoders' Unicode escaping, U+2028/U+2029 and normalization contracts are **not** yet a universal CPE canonical-identity standard. Keep the v0 research namespace, harden UTF-8/Unicode and benchmark different runtimes before freezing.
- Proposal claims are not concept semantics, language equivalence, policy verdicts or legal rights. An actual acceptance needs independent native-language source grounding, counterexamples, deduplication, heldout consumer tests, owner decision and immutable new accepted authority release.
- All PRs remain drafts; no merge, private GHA spending, public/private transfer, release or deployment approved.
