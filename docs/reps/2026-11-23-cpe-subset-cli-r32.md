# CPE subset CLI R32 — portable research-only consumer adapter

**2026-11-23 · DRAFT / RESEARCH_ONLY / PUBLIC-SAFE SYNTHETIC DATA ONLY**

Governing authority: private CPE #348 (federation), #202 (consumer feedback), #201 (semantic authority), #79 (independent gates). This is an **add-on to the existing cpepack Go source** under the public Explorer research PR stack, not a new lock, concept authority, server or accepted KnowledgeRelease. Required parent: public Explorer PR #47 at `ba78aec76f143df3bd2beb5ea42f5430503f4a09`.

## Contract and command

The existing `cmd/cpepack` package build remains the default. New optional arguments expose the *existing* `cpepack.BuildSubset` and `cpepack.BuildPinnedSubset` functions, **without bypassing the package content/lock validation performed by Build**.

```sh
# Legacy research package projection (unchanged):
go run ./cmd/cpepack -packages testdata/cpepack/packages.json -lock testdata/cpepack/lock.json -root test.public.literature

# Opt-in research-only subset inspection; requires explicitly supplied selector and references:
go run ./cmd/cpepack -packages testdata/cpepack/packages.json -lock testdata/cpepack/lock.json -root test.public.literature \
  -selector selector.json -representations representations.json

# Independently pinned research-only view; both expected digests are mandatory:
go run ./cmd/cpepack -packages testdata/cpepack/packages.json -lock testdata/cpepack/lock.json -root test.public.literature \
  -selector selector.json -representations representations.json \
  -expected-source-sha256 64-lowercase-hex -expected-selection-sha256 64-lowercase-hex
```

This command **does not supply its own trusted pin source**. In real use, a separately admitted release/consumer owner must furnish both expected digests. The tests derive their pins from synthetic fixtures solely to exercise the comparison operation; they are *not* acceptance evidence for independent issuer approval. The input schema versions remain research-only `cpe-lock-v0`, `cpe-representation-ref-research-v0`, and `cpe-subset-research-v0`. The direct Go library and CLI use the same in-process engine; **that is one implementation, not two independent consumers**.

## Three audiences

- **Humans:** run locally, inspect indented read-only JSON and stable error strings. No network, artifact hydration, policy enforcement or source promotion.
- **Automation:** exit 0 for verified *research* output; 1 for semantic/lock/pin refusal or output failure; 2 for malformed input or incomplete command. Required inputs are bounded at 2 MiB each and unknown JSON fields are refused.
- **Agents:** consume `authority=research_only`, source-lock and subset SHA-256 fields, machine-readable concept IDs and representation refs. A success status alone is **not** a release authorization or permission to fetch/distribute media.

## Red/blue and source qualification

The R32 test file includes: legacy CLI/direct Build parity; exact subset CLI/direct BuildSubset parity (including required-edge concept closure and representation metadata); both independent pins matching; wrong source or selection digest refusal; partial selector/refs/pin refusal; public subset refusing a private reference. Existing tests continue to own deeper identity, source lock, private transitive closure and R26/R27 portability.

Run in the **public-safe** workflow `cpe-lock-source-qualification.yml`, which exercises offline `go test -count=2`, race tests, vet and Linux/amd64+arm64 crossbuild. Collect source SHA, action run ID, first failure and clean status. No private CPE data may enter public CI.

## Explicit holds

No semantic/fact promotion, S4/S5 private adoption, signed release authority, source rights/MAF byte custody, runtime network query, independently implemented second consumer, user-hosted deployment, merge, accepted release or private GitHub Actions spend. Source revision and visibility semantics must not be guessed from artifact URL or model similarity. Next: independent reviewer against exact PR head; then *different* consumer read-model integration under #202 with user-approved accepted snapshot after #259/#79.
