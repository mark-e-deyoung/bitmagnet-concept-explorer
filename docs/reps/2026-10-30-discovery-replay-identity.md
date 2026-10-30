# Discovery proposal replay identity receipt

**Scope:** source-only synthetic review at the public source commit
`95e7fb3c34b68408173ce0e743b42d8af10752e9`. No corpus, CPE, network service,
Foundry, release, or deployment action was used.

## First failures

- The workstation has no `go` executable (`go: command not found`). Tests were
  run offline in the cached, digest-pinned Go 1.27.1 builder instead.
- The new conflicting-ancestry regression initially failed: two records with
  the same `source` and `evidence_ref` but different `ancestry_ref` values were
  silently treated as duplicates. The first record won, so reversing input
  order could select different ancestry and proposal identity.
- The initial implementation also exposed that the older
  `/api/v1/discovery-proposals/preview` and newer
  `/api/v1/sweeps/discovery/preview` envelopes have different identity inputs.
  The older envelope has no source, language, projection, or ancestry fields;
  their identifiers are not interchangeable.

## Provisional research-v1 normalization

For the sweep proposal envelope only:

1. Identity includes a version marker, proposal kind, trimmed candidate label,
   lowercased language, concept target, projection digest, and normalized
   evidence records.
2. Evidence records are sorted by source and evidence reference. An exact
   duplicate `(source, evidence_ref, ancestry_ref)` is collapsed, making exact
   replay and permutations idempotent.
3. Repeated `(source, evidence_ref)` with different ancestry is rejected. The
   implementation does not choose which ancestry is true.
4. Different evidence and ancestry remain represented in the hash. The
   resulting `cpe-discovery-proposal-research-v1` / `bce-sweep-v1-…` identity
   is local replay identity only.

This is a provisional consumer-side envelope normalization. It does not assert
that a digest verifies source authenticity, that evidence is independent or
semantically correct, or that a proposal is a CPE record. The older preview
route retains its separate legacy identity contract pending an owner decision
on a common envelope. No proposals are transmitted, persisted, accepted, or
promoted. CPE #201/#202 remain authoritative for projection and discovery
contracts and all semantic adjudication; this change makes no claim of CPE
compatibility or acceptance.

## Verification and disposition

At this worktree's resulting commit, offline in
`golang:1.27.1-bookworm@sha256:8d48e12ec56735e9358640898b9d9b9fcca110612ed8a5567438c0a1baa24e66`:

- focused `go test -count=1 -run ResearchProposal ./...`: PASS;
- `go test -count=1 ./...`: PASS;
- `go test -race -count=1 ./...`: PASS;
- `go vet ./...`: PASS;
- `gofmt` check: PASS.

Fictional hash fixtures only. Commit is local to the isolated review worktree;
not pushed, proposed, released, uploaded, or deployed. Next decision: CPE
owners determine whether the legacy endpoint should migrate to a shared,
versioned consumer envelope; until then, consumers must not compare IDs across
those routes.
