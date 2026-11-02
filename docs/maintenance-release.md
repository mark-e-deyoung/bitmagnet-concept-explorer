# Maintenance qualification

The published baseline is v0.0.2. `VERSION` selects the next untagged maintenance candidate. This work changes artifact identity and repeatability, not read-only product authority or research acceptance.

## Build and verify

From a clean checkout, run `python3 scripts/release/build_image.py --evidence build/artifact-identity.json`. The helper requires a full Git revision, safe candidate version and digest-pinned official Go builder. It makes one build attempt, verifies OCI version/revision/source labels and records the local image ID/platform. It never pushes an image or deploys a host. An unavailable reviewed builder digest is a failure, not permission to retry a floating tag.

The existing public Actions jobs run Go unit/vet/race, amd64/arm64 builds, browser/axe qualification and the disposable Foundry F0-F3 ladder. Browser dependencies use the committed `package-lock.json` and `npm ci`. The lifecycle receipt includes `artifact-identity.json`; successful endpoint checks alone do not establish source identity.

## Bringup status

- F0-F3: prior v0.0.2 evidence exists; this candidate must get its own public source/lifecycle receipts.
- F4 physical TrueNAS canary: held for a separately allocated non-production target.
- F5 official catalog admission: held after F4 and upstream review.
- Research package/graph functionality remains research-only. This candidate does not accept private semantic or policy authority.
- Publication/tagging, registry push and actual deployment remain separate owner decisions.

The catalog fixture describes an untagged candidate image. No image with that tag is claimed to exist in a registry. Public qualification builds it only in the disposable runner and removes its containers/networks afterward.
