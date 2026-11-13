# CPE Workbench — immutable Git package lock source REP 001

Status: **research-only, synthetic, source qualification**. Ownership: private CPE #348 (federation), #201 (semantics), #202/#229 (sweeps), Bitmagnet product #18 (Explorer), Foundry private #274 (app lifecycle). No accepted Plane A/B/C data, owner promotion, private/public data movement, merge, release, publication, OCI push or TrueNAS deployment.

## Reuse and separation

This source workcell adds one offline Go stdlib library and CLI to the existing public Explorer repository. It does not replace the pinned `bce-graph-preview-v1` loader, sweep-id-v1 identity, read-only API/UI, CPE proposal rehearsal, or Foundry. The candidate formats are `cpe-package-v0`, `cpe-package-id-v0`, `cpe-lock-v0`, and `cpe-readmodel-v0`; they are intentionally distinct from pre-existing sweep identity/schema IDs.

A package carries an origin repository, exact 40-character Git commit SHA, SHA-256 digest of its canonical concept content, visibility, research-only authority state, and explicit dependencies. The lock separately pins origin, versions, digests, scope, authority and dependencies. `Build` validates the requested root transitive closure, detects cycles, incompatible IDs/versions, missing/stale pins and semantic-definition collisions, and emits a deterministic, read-only JSON projection. Canonicalization sorts concept IDs, lexicon entries and typed edges and uses Go JSON struct field order, **not** the separate sweep-id-v1 Python/Unicode canonicalizer. Identity compatibility across Go/Python must be independently proven before adoption.

Public materialization rejects any private transitive dependency, and a private package elsewhere in the same federation lock does not affect the selected public output or its digest. Research-only labels are retained; no automatic fact or semantic acceptance, and no source text acquired.

## Native local test receipt, 2026-10-31

Tested isolated source snapshot with `go1.23.2 linux/amd64`, with network-independent commands:

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test ./...
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go vet ./...
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test -race ./...
```

All commands PASS (source-only local runner), including public closure reproducibility, unrelated private package isolation, multilingual term homograph nonmerge and eleven adversarial fail-closed negatives. Source blob IDs checked independently with `git hash-object` against remote GitHub contents:
- `internal/cpepack/pack.go`: `2e90ce1bc7e6b8bf39d9d9fe2cd3e53b44d6d415`
- `internal/cpepack/pack_test.go`: `183891cb589aa7f4b14ed259373a4f120b9d0ff0`
- `cmd/cpepack/main.go`: `ec547db82a150a1581f7f92ed620ec857969c9b3`

**Limits and red-team debt:** repository SHA is syntactically pinned but origin commit existence/signature not offline-attested; package manifest itself is bound by Git SHA and lock fields but not signed; JSON strict decoding/size bounds are CLI-only, not library DTO limits; no extra-safe I-JSON cross-language canonicalization proof; no actual Git checkout resolver, signed release lock, schema-migration alias, source-rights adjudication, SQLite/FTS or OCI layout yet. This workcell must not be called a federated accepted release or a TrueNAS F0–F5 pass.

## Next source-safe admission

1. Independent check of this draft's exact head: local/source CI with Go 1.22 compatibility, gofmt, race, vet, architecture crossbuild; run negative mutation tests and a separate independent Go/Python canonicalization fixture.
2. Add explicit self-contained synthetic `base-public`, `literature-public`, `private-domain` JSON packages/lock and two-process identical digest replay. Test namespace ownership, old v1 schema-ID alias collisions, source-lineage and public diagnostic redaction.
3. Read-only projection adapter to the existing Explorer UI/API/CLI/agent paths, one exact pin with parity and screen-reader/keyboard evidence; do not create a second controller.
4. Add research sweep saved-config CRUD/prepare/receipt and repair #229 cross-scope bare `query_id` collision (composite scope/track/query key) **without** breaking existing IDs. Keep blind A/B/Stage P HELD.
5. JSON vs SQLite/FTS vs OCI-layout lowering on the *same* digest-locked fixture; Foundry source render and exact-version F0–F5 only when admitted.

Git/PR = source and tests; owning DLE = authority and qualification baton; only independently approved, large binary deliverables go to owner-controlled Drive with SHA-256/readback. Nothing here grants semantic promotion or public release.
