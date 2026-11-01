# v0.0.1 release review: READY source-only corrections

**Authority:** `mark-e-deyoung/bitmagnet-tools#18`, independent-review baton in comment `6533628197`.
**Immutable release source:** `95e7fb3c34b68408173ce0e743b42d8af10752e9` (`v0.0.1`).
**Source-code head after privacy-safe author rewrite:** `c2d3e4b39e82aa35950b144b6985984fa1e11c93`.

This receipt covers only the source-only regression/correction lanes authorized by the baton. It is not release, CPE, Foundry, catalog, live-corpus, or deployment approval.

## Lane A — SearXNG quota admission

- First failing regression: concurrent invalid language/query requests reserved three hourly slots; a pre-canceled request reserved one. Neither case made a network request. A cancellation during the pacing wait also left the old code's reservation behavior observable.
- Correction: validate context, language, and query before acquiring service admission; pace first and commit an hourly slot only when ready to proceed. Canceled pacing waiters do not consume request quota.
- Preserved controls: opt-in use, two concurrent slots, two-second spacing, three requests/hour per endpoint, and 429 refusal cooldown.
- Lane worktree commit before author privacy rewrite: `aee9e10a0dec31ad27013ca66c18de6683ebe091`; privacy-safe integrated commit: `f6b13e6dac14452be01c93c391cd02e2c59db748`.

## Lane B — sweep proposal replay identity

- First failing regression: duplicate `(source, evidence_ref)` entries with conflicting ancestry were silently deduplicated, so input order selected the retained ancestry.
- Correction: sweep preview now emits a `cpe-discovery-proposal-research-v1` identity; normalizes evidence order, collapses exact duplicates, rejects conflicting ancestry, and keeps changed ancestry identity-distinct. The legacy preview identity remains separate because its envelope has fewer provenance fields.
- This is a local replay contract only. It does not verify evidence, create an accepted CPE record, transmit proposals, or establish compatibility with CPE. Issues `content-policy-engine-private#201/#202` retain authority.
- Lane worktree commit before author privacy rewrite: `187d64c9182c7b19d20ed5c456efb9ad027384b4`; privacy-safe integrated source-code commit: `c2d3e4b39e82aa35950b144b6985984fa1e11c93`.

## Verification

All Go commands used the cached builder `golang:1.27.1-bookworm@sha256:8d48e12ec56735e9358640898b9d9b9fcca110612ed8a5567438c0a1baa24e66`, with networking disabled and Go module proxy/checksum downloads disabled.

- Focused A and B regression tests: PASS.
- At integrated source-code head `c2d3e4b39e82aa35950b144b6985984fa1e11c93`: native/amd64/arm64 builds, full unit tests, race tests, vet, and gofmt: PASS (`scripts/qualify_local_garm.sh`). The later documentation-only receipt commit was exact-head qualified before privacy-safe commit identities were normalized. After rewriting the unpublished branch to the account noreply identity, the source-code commit was requalified at this new SHA; qualification passed.
- A pinned linux/amd64 OCI build at source-code head `c2d3e4b39e82aa35950b144b6985984fa1e11c93` passed with SBOM/provenance. The OCI archive SHA-256 is `a8aa8f990eb08d2abc58de6ec6802325d898a2a0db4031766cd72d00af29fcf4`; OCI index `sha256:df421099cf21686995c921c7cd8f92bd4ce8988537de756075ec0d973654547d`; amd64 manifest `sha256:82ea6cee4cf51af9301c4adbf5152aeac7d04f70253542fa1cb9b092b1e2c7c9`; attestation manifest `sha256:338837652acc640c0f3cfbef241910446245bdbe144eff3ab5587628e56e350d`. The archive remains at `/tmp/bce-review-c2d3e4b.oci.tar`; it is local verification evidence only.

Qualification setup first failed because the read-only container mount did not include the worktree's shared Git metadata; an earlier attempt also mounted `/tmp` without executable permissions. The corrected run mounted the worktree and shared metadata read-only, used an executable temporary filesystem, and passed without source mutation. The first Git push was rejected because commits used a private email; the unpublished branch was rewritten to the account noreply identity and requalified.

## Distribution check and artifact disposition

- Downloaded the public v0.0.1 OCI archive without GitHub authentication. Independent SHA-256: `e0fcc968e5dc819d2d75a42096c99dd60598790a0908da0b8bcc426056273d3d` (3,354,112 bytes), matching the release asset.
- Independently hashed the archive's OCI index blob and confirmed index `sha256:0cf08c60b256d1335200a5276a0b68e6feaf6e8ad879ca125bc9ca11ce85d07c`, amd64 manifest `sha256:8173e91e677ef919e7ec1acc40e678e488629772f64c58a85e052d09b2dd19bc`, and attestation manifest `sha256:c76a0fee0ca61b1d36a4302ab2b3951020b0b156364854ba7fb8bdea1c55182c`.
- Anonymous GHCR manifest access returned 401 both before and after obtaining the registry's anonymous pull token. The image remains private/not publicly pullable; no cache-authenticated Docker pull is counted as anonymous evidence.
- The existing v0.0.1 tag, release asset, image digest, and catalog candidate were not changed. The review-branch OCI archive remains under `/tmp/bce-review-c2d3e4b.oci.tar` for local verification only; it was not pushed, attached to a release, or uploaded to Drive.

## Holds and next executable baton

No live Bitmagnet canary, TrueNAS operation, Foundry materialization, CPE promotion/update, official catalog submission, merge, retag, or new release was performed. The CPE data loader remains a digest-pinned provisional preview without a governed update/rollback/readiness lifecycle. Foundry candidate `#347` remains unadmitted; exact target profile, accepted projection, and disposable lifecycle evidence are still absent.

**Next:** review the draft source PR from `review/baton-ready-fixes`; independently accept or revise the two regression fixes. Separately resolve GHCR public access or choose an approved publicly fetchable immutable artifact mechanism. CPE owners must define/accept the projection and intake contract before any CPE update workflow is implemented. Foundry owners must admit the exact target profile before lifecycle work. Preserve all existing owner/security/release/deployment gates.
