# $0 live Bitmagnet catalog and public CPE research rep — 2026-10-31

**State:** source/draft experiment, NO CPE authority acceptance, no release or local site deployment.

## Inventory found — correct public-CPE interpretation

- Current concrete public Git-backed CPE pilot: [draft public PR #6](https://github.com/mark-e-deyoung/bitmagnet-concept-explorer/pull/6), `testdata/public-cpe-packs/gutenberg-lore-v0/`, root source SHA `77e12615fe6a0f0c4b431ff360e22da3f728ffcb`.
- `graph-preview.json`: 10 public concept **types**, English/German/Thai/Japanese/Spanish lexical labels. Examples: literary work, fictional character, edition, illustration, source record, sourced claim, rights claim. This is `bce-graph-preview-v1`, not the canonical accepted schema.
- `catalog-evidence.json`: four sourced eBook/work records (Project Gutenberg 62/55/11/35), characters/worlds, example claims, related artwork *references* (no binaries, no rights conclusions).
- **No separate standalone, accepted public CPE repository** verified in available `mark-e-deyoung`/SemperSupra GitHub inventory at this time. Public work **does exist**, but so far as the draft Gutenberg research pack. The future public/private domain Git package federation/compatibility is separately owned by private CPE [#348](https://github.com/mark-e-deyoung/concept-policy-engine-private/issues/348); accepted A/B/C semantics by [#201](https://github.com/mark-e-deyoung/concept-policy-engine-private/issues/201) and discovery/intake [#202](https://github.com/mark-e-deyoung/concept-policy-engine-private/issues/202).
- Other candidate domains (hardware/DE2-115/Jetson, game consoles/games, anime/characters) **are permissible public research topics where individual source records are releasable** but are not silently asserted as existing qualified/merged domain packs.

## Actual real-system public Actions rep

This draft stacks on the **existing #6 branch**, reuses without rewriting the **already-real v0.10.0 Bitmagnet + Postgres + BCE** original public GHA harness in [#16](https://github.com/mark-e-deyoung/bitmagnet-concept-explorer/pull/16), and adds only a bulk catalog reader and live injection/qualification.

1. On disposable public standard GitHub Actions runner, pull **digest-pinned real Bitmagnet v0.10.0** and real PostgreSQL; **no mock** GraphQL service, no DHT worker, no TMDB, no torrent downloads, no public service port.
2. Use Project Gutenberg's [official offline bulk catalog](https://www.gutenberg.org/ebooks/offline_catalogs.html), weekly `pg_catalog.csv.gz`, with **one bulk download**, no page scraping, bounded size/time, gzip SHA-256, and source row count. If the primary official URL fails, try the official development mirror once. Preserve true catalog titles, author strings, source record IDs, language and Gutenberg release metadata in the machine-readable manifest.
3. Deterministically select a stratified **~232** book catalog records (EN/DE/FR/ES/other) plus four mandatory known anchors #62, #55, #11, #35, with an unchanged seed; do **not** ingest text/artwork. Fields `infoHash`, `size` and the marker prefix in `name` are explicitly **synthetic disposable import identities** — no real torrent, peer or payload exists. These metadata records are not a license/rights determination, and the Gutenberg eBook record is not necessarily the original edition/work.
4. Seed the actual isolated Bitmagnet database through its genuine `POST /import`. Test Bitmagnet GraphQL native FTS and actual Explorer API and CLI, and preserve the prior five synthetic negative/overlap records and native AND/phrase distinction. Test real CPE graph file SHA pin, `GET /api/v1/concepts` type count, bilingual labels and authority status. Use public catalog evidence to formulate an **unverified Barsoom → A Princess of Mars lookup hypothesis**, and verify its actual indexed Gutenberg #62 record via native Bitmagnet search.
5. Runner self-destructs; no private data, GitHub write token, CPE promotion, release artifacts, GHCR publication, Foundry install or site apply.

**Cross-cutting negative:** An invented `infoHash` from a book record is not a real torrent. The product should never issue or initiate peer interactions from these test identifiers, and may not label these as real torrents, validated sources, source rights or high-confidence concept assertions.

## Reproducibility and disposition

- Seeded extraction algorithm, immutable Bitmagnet and PostgreSQL OCI digests and small code/fixtures are public Git text. The exact source bulk catalog `sha256`, selected record identities, sample totals and native execution outcomes are recorded in the Actions job log and manifest on the ephemeral runner.
- The weekly feed is **mutable**. A PASS at a particular SHA pins that one observation, not universal repeatability when Gutenberg updates its catalog. If repeated reps require bit-exact input replay, qualify a small, rights-safe frozen metadata subset with manifest SHA before promotion. The catalog full gzip is **not** committed or uploaded to Drive.
- Acceptance: native source 4/4 and real-system run each terminal PASS at the same current head, independent log readback, no accepted CPE authority claims. HOLD if feed cannot be obtained, parsed, or the real Bitmagnet importer/indexer cannot serve a known anchor.
- An offline source/mock alternative is **not** used to satisfy this runtime qualification. No private TrueNAS or actual operator corpus HIL is claimed.

## Next post-rep work

Independent review of provisional pack types and actual recall/false-linkage on held-out queries; decide whether public CPE moves from research folder into a separately owned public Git package/repo under CPE #348. Source qualification alone does not approve public CPE ontology or normative concept IDs.
