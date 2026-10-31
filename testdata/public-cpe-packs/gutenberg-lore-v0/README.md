# Public CPE research pack: Gutenberg lore v0

**Status:** publishable-source **research candidate only**. Not an accepted Concept Policy Engine snapshot, certified fact collection, distribution permission, production asset bundle, or Bitmagnet catalog classification. The **concept graph** is compatible with the Explorer's existing `bce-graph-preview-v1` loader (max 128 nodes, 2 MiB, SHA-256 pinned); the **catalog-evidence** JSON is a separate future fact/entity-interchange experiment, **not yet loaded by the production Explorer**.

## Objective

Prove a small, real, public-source concept subset can run on free public GitHub Actions and the same single-container/TrueNAS source build without exporting any private CPE materials. Contrast:

1. **Concept vocabulary:** public `graph-preview.json` (10 basic reusable types including literary work, edition, character, setting, illustration, rights statement and cited assertion), with EN/DE/TH/JA/ES lexicalizations. These **define meanings**, not particular book or person identities.
2. **Entity/source evidence:** public `catalog-evidence.json` (four Project Gutenberg reference books, sourced relationship candidates, three specific historical-art references, and seven authoritative or licensed-media source discovery endpoints). This is **source-reported metadata**, not independently accepted facts or universal reuse permission.
3. **No source binaries:** not a downloader, crawler, media repository or content acquisition workflow. No ebook text, cover images, artwork, credentials or protected CPE graphs.

## Small public work sample

- Project Gutenberg eBook [62](https://www.gutenberg.org/ebooks/62): Edgar Rice Burroughs, *A Princess of Mars*; series and original catalog subjects explicitly mention Barsoom, John Carter, Dejah Thoris.
- eBook [55](https://www.gutenberg.org/ebooks/55): L. Frank Baum, *The Wonderful Wizard of Oz*.
- eBook [11](https://www.gutenberg.org/ebooks/11): Lewis Carroll, *Alice's Adventures in Wonderland*.
- eBook [35](https://www.gutenberg.org/ebooks/35): H. G. Wells, *The Time Machine*.

The *work* is not its Gutenberg eBook manifestation or the same as a 1917 illustration, modern film, modern translation, stage production, actor, or model. Provenance-bearing edges are candidates until independently qualified.

Concrete related-art leads:
- [1917 Princess of Mars Plate 2 — Frank Schoonover](https://commons.wikimedia.org/wiki/File:Princess_of_Mars_Plate_2.jpg), Wikimedia Commons item record with US public-domain claim.
- [1900 W. W. Denslow Oz illustration](https://www.loc.gov/item/98508888/), Library of Congress item record with a rights advisory.
- [1900 illustrated *Wonderful Wizard of Oz* edition](https://www.loc.gov/item/03032405/), Library of Congress digitized original edition with collection-level reuse statements.

No art files are fetched, cached or redistributed, and **specific item rights remain unqualified** by this pack. Illustrated and adapted editions can contain **independently copyrighted components**. US public-domain status is not automatically a global-rights conclusion. A source claim about copyright, license or territorial reuse is represented as evidence; downstream systems evaluate what uses are allowed.

## Acquisition path (separate future implementation)

Use Project Gutenberg's **daily bulk RDF/XML metadata** and (where appropriate) weekly CSV/MARC metadata via [official Offline Catalogs](https://www.gutenberg.org/ebooks/offline_catalogs.html), rather than website scraping. Preserve metadata snapshot, origin, source ancestry, work/edition split, original title/script, subject, date and source claim. For related art, query item-level catalog metadata from Wikimedia Commons, Library of Congress, Smithsonian Open Access, Openverse, or The Met, with optional Standard Ebooks edition links; preserve per-item attributed rights assertions and candidate relationship confidence. Do not assert artist-style/character likeness or direct adaptation from generic keyword co-occurrence.

## Reproducibility and tests

- `public_gutenberg_pack_test.go` checks the exact graph using the existing application's SHA-256 pin and rejects wrong digests.
- It also verifies four distinct Gutenberg source IDs, non-duplicate sourced claims, no orphan related artwork, and **no downloaded or retained media**.
- Source-only standard GitHub Actions (stacked experimental public workflow) runs these tests, Go vet and native architecture builds, with **no private CPE source, secrets, TrueNAS corpus, real peer data or image downloads**.
- Run locally or in a public CI runner: `GOPROXY=off GOSUMDB=off go test -run 'TestPublicGutenberg' ./`. Run all tests and race suite in the inherited public cloud qualification workflow.
- UI/API concept preview can load `graph-preview.json` only with `BCE_CPE_GRAPH_FILE` and `BCE_CPE_GRAPH_SHA256` set together. It remains `pinned-preview-not-normative`. The actual TrueNAS app wizard does **not yet** mount these values; Foundry's separate app package workstream owns that correction.

## Earn-its-keep tests

Compare (A) a plain title keyword lookup to (B) this public concept/source candidate pack on held-out, independently reviewed creative-project questions. Required cases: A Princess of Mars work ≠ Gutenberg ebook release ≠ 1917 illustration; translated/alternate names; Oz historical art ≠ 1939 motion-picture art; source copyright claims scoped to **US/item** and uncertainty; related-source candidate ≠ verified direct work adaptation; one mirrored catalog claim ≠ independent corroboration. Keep the pack only if it provides a reproducible improvement in correct work/character/art discovery, reviewer effort, or deployment/CI qualification at tolerable maintenance cost.

**Other domain candidates:** game consoles/games, NVIDIA Jetson and Terasic DE2-115 (technical metadata) and anime/characters (e.g. Fairy Tail) are independently useful, but should be added as separate small packets *only after this first contract earns its keep*. Copyrighted character metadata may be indexed with provenance; no assumption of rights to artwork/characters or indiscriminate ingestion of unrelated sources.

## Authority and custody

CPE owns accepted concept and fact qualifications. This public pack is independently versioned research input and must never silently promote `status: source_report_not_independently_qualified` to accepted truth. Source code and small public fixtures stay in Git, results/acceptance and handoffs in the authoritative DLE, permitted larger binaries in controlled Drive under existing provenance/custody methods. This pack contains **no media binaries**.
