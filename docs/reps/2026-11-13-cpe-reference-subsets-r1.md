# CPE reference/subset R1 — original synthetic research (2026-11-13)

**Status:** DRAFT / SOURCE-SAFE / RESEARCH_ONLY. Governing CPE DLE [#348](https://github.com/mark-e-deyoung/concept-policy-engine-private/issues/348); semantic boundary #201/#259, Foundry private acquisition #6 and existing Foundry OCI/native-cache contracts. This is an independent **stacked child** of Explorer draft #23 at exact parent commit `13bbfc5809e5e46f798af78ed0d6b373a214d98e`, **not** a migration or acceptance of private CPE concepts. Do not merge, tag, publish, adopt or deploy without separately authorized gates.

## Objective and design

Extend the existing **research-only** `cpepack` package-lock projection with **typed external representations** and a deterministic **semantic subset**. A concept remains present when all its associated media are absent. Existing package identities `cpe-package-v0/cpe-lock-v0/cpe-readmodel-v0` and `sweep-id-v1` are **unchanged**; this rep adds `cpe-representation-ref-research-v0` and `cpe-subset-research-v0` without modifying the existing `Build` input/output or approved authority. New files only: `internal/cpepack/representation.go` and `representation_test.go`.

Representation refs are metadata-only and cannot fetch files, verify bytes, acquire credentials, select real model runtime, adjudicate rights or promote semantic authority. Identity kinds remain distinct: `oci` (bundle digest), `content-manifest` (MAF local exact file manifest) and `raw-sha256` (single raw member). A locator is not a digest; no mutable URL/local path is part of the v0 public record. A ref includes concept ID, typed role, modality/format, source revision, visibility, rights, external digest and optional source evidence/dependency IDs.

The `SubsetSelector` selects root concepts plus **explicit required edges already present in the source projection**. The result retains out-of-subset optional semantic edges, includes required representation dependencies, and marks representation dependencies from outside the selected concept set. Public projections reject direct or transitive private/restricted representation refs. Unrelated private example refs do not enter the public subset. Refs sharing byte digest remain independent **semantic** bindings. A view's SHA-256 covers source package-lock digest, normalized selector, selected concepts, out-of-subset relations, references and view-version/authority. **Not an accepted knowledge release or signed artifact.**

## RED -> BLUE -> independent verification

- Initial RED synthetic `TestArtifactDependencyCycleRejected` found missing cycle rejection (returned success instead of `representation_dependency_cycle`); preserve initial failure as local study evidence. BLUE introduced tri-color traversal for required representation dependencies. Retest GREEN.
- Local temporary Go 1.23.2 linux/amd64, Go 1.22 module, **isolated exact-source module** with a _test-only local stub_ for the existing `cpepack` types/helpers (the stub is NOT committed). Commands: `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test -tags local_stub -count=2 -shuffle=on -v ./internal/cpepack`, `go test -tags local_stub -race -count=1 ./internal/cpepack`, `go vet -tags local_stub ./internal/cpepack`, and Linux/ARM64 test cross-compile: PASS. Core tests cover reference/subset determinism, old-source replays, required concept closure, out-of-subset links, missing-media independence, same bytes/different meaning, cross-concept artifact dependencies, explicit source authority version, direct/transitive private data, rights, wrong-kind digest, invalid metadata, missing dependencies and cycles.
- Source blobs independently matched local Git objects after GitHub publication: `representation.go` `7df02c9025dc54a96bc234ecd6ea16c1c7c63111`; `representation_test.go` `a7377e77be09ef56b12dba4c833bbe79c40977af`. Full repository GHA **must independently execute the exact draft head**; local stub build does not qualify the integrated source.

## Limitations and next falsifiers

1. **Trust boundary**: `MaterializeSubset` receives an existing `Projection` that MUST originate in an independently verified `cpepack.Build`; a caller-controlled copied `Projection` is not cryptographically authoritative. Prove exact source/lock binding across integration and consider a safe factory/wrapper, not unreviewed acceptance.
2. **Synthetic only**: no real CPE accepted A/B/C release, source-rights verification, private media, real WAV/BVH/KML, MAF receipt validation, native cache hydration or protected host. Reference state is not asset availability.
3. **Hardening next**: cross-language canonical JSON identity check; bounded/strict JSON admission; source-assertion/occurrence loss check; optional-edge leakage and rights exhaustion; user-driven snapshot selector and broader CPE facts; concurrency/corruption/GC in MAF-owned cache. Retain original negative vectors independently (not the treatment-only suite).
4. **Efficiency**: no daemon, registry, extra cache, vector database or backend migration. Synthetic input sizes and first-failure measured cloud receipts are required before any SLO.

## Three-audience DX

The same deterministic JSON projection can be used by human Explorer, automation CLI and an agent through existing read-only `cpepack` front ends; this workcell does **not** claim UI/API parity until a separate integration rep. Failure errors are stable machine-readable tokens; do not infer authority or legal distribution rights from a PASS.
