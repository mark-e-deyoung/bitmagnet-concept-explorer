# Public cloud source-only qualification — 2026-10-31

Status: **RESEARCH-ONLY, NOT A PRODUCT FEATURE**. This work is stacked on the current open draft corrective source head, is not merged, and is not a CPE accepted ontology, CPE fact store, or product deployment.

## Reproducible purpose

- Independently exercise the latest corrective BCE Go source using public standard GitHub Actions runners with a synthetic-only, secretless workflow.
- Compare a minimal source-scoped fact/evidence filter against unsupported assumptions about Boolean NOT, implicit negative facts and false role/identity joins.
- Retain exact source commit, job/run/attempt identifiers and failed-test logs from GitHub's native execution receipts. A workflow config alone is not an execution receipt.

## Synthetic experiment design

**Comparator:** ordinary word-match/exclusion cannot distinguish missing metadata from refutation; no direct field/role/referent evidence. **Experimental primitive:** a *separate* tiny typed evaluator of qualified source claims (not integrated into BCE service or CPE authority).

Fictional fixtures challenge four evidence states: supported, refuted, conflicted, unknown; three query operators: require, exclude_observed, require_absence; mirrored ancestry; multiple citizenship values; parent vs subject facts; work tag vs scene depiction; appearance observation time; malformed inputs; nonqualified observations. No real performers, titles, private corpus or provider credentials are included.

**Safety/correctness invariants:** absence of a record never becomes a qualified negative fact; disagreement is not majority-voted away; source mirrors do not inflate independent evidence; no transitive identity/participation/citizenship claims; unqualified assertions cannot satisfy positive or strict-negative query. A source's raw lexical NOT remains distinct from semantic exclusion.

**Kill/simplify comparator:** if the experimental evaluator adds no measurable correctness benefit over existing CPE generic assertions, retain the generic contract. Do not add a factual database, full reasoner, face model or independent search index on the strength of these fixtures.

## Public runner envelope

No secrets, no external search, no Bitmagnet endpoint, no CPE projection, no unreviewed private source, no privileged containers or paid/larger runners. Source Go module declares no third-party dependencies. Public standard ubuntu-24.04 hosted runners; pinned checkout and setup-go actions; Go toolchain, stdlib-only offline module/test/vet and native cross-build; Node UI-state regression where present. Restrict workflow token to contents:read, use bounded timeouts and no release/deploy permissions.

Expected independent jobs: (1) native unit/vet/format/Node; (2) Go race; (3) linux/amd64 + linux/arm64 builds. A test PASS proves the source fixture and test runner, not live source coverage, catalog materialization, real-person accuracy or physical TrueNAS compatibility.

## Limits requiring other evidence later

- Real private Bitmagnet accumulated corpus, actual coverage, source facets and network adjacency require approved local read-only test under its separate authority.
- Private CPE accepted projections, source ancestry and any protected test cases remain private; public code accepts only synthetic-compatible contract examples.
- Official TrueNAS materialization/upgrade/rollback and local networking need disposable pinned TrueNAS HIL under Foundry, never inferred from Go unit tests.
- External indexes/person/catalog APIs require independent source rights, availability and query-consent qualification; synthetic stubs are not proof of live results.
- P2P piece sampling requires separate peer-transfer admission, network/rights/resource controls, and is not enabled by this research.
- GitHub Actions runs may be disabled, require approval, lack a pinned toolchain, or be blocked by repository policy; record BLOCKED and the exact missing gate rather than claiming cloud PASS.

## Next decision

After the native runner receipts arrive: record exact SHA and first failures, fix only bounded source/test defects without touching earlier actors' draft PRs, then assess KEEP / SIMPLIFY / DEFER / KILL. Expansion into production BCE code or CPE accepted authority requires a separate governed decision.
