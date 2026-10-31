# OCI research read-model projection rep

This source-only v0 experiment serializes the deterministic cpe-readmodel-v0 projection as a **non-executable OCI data artifact**. The serializer follows OCI image layout/index/manifest and the empty config + artifactType pattern for non-container OCI artifacts (OpenContainers image-spec image-layout.md and manifest.md). It produces only a local directory, not a runnable image, OCI push or registry publication.

CLI (after qualifying parent PR #23): `go run ./cmd/cpepack -packages testdata/cpepack/packages.json -lock testdata/cpepack/lock.json -root test.public.literature -format oci-layout -out /new/path`.

The public-only dependency resolver runs before any filesystem output. Exact inputs produce byte-identical manifest/index/digests across two runs; package iteration order has no effect. A private-root attempt produces no directory. Existing destination paths are refused rather than overwritten. Original source, lock, Git revision, qualified rights and acceptance remain separate authority; this is synthetic-only and research-only. The writer does not attest actual remote Git object existence or independently verify rights, OCI registry compatibility, ORAS transfer, signed provenance, SQLite performance or Foundry F0-F5. Those are later gates.

Sources: https://github.com/opencontainers/image-spec/blob/main/image-layout.md and https://github.com/opencontainers/image-spec/blob/main/manifest.md .
