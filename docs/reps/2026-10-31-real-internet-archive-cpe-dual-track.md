# Public Internet Archive → real Bitmagnet + SearXNG → CPE research roundtrip

**Purpose and authority:** Integrate independent real-system [Explorer draft #19](https://github.com/mark-e-deyoung/bitmagnet-concept-explorer/pull/19) (genuine pinned SearXNG + Bitmagnet/Explorer live paired sweep) with independently native-qualified [draft #20](https://github.com/mark-e-deyoung/bitmagnet-concept-explorer/pull/20) (real Internet Archive metainfo from item-license-indexed official API). This is an **isolated stacked draft**, not a merge into main or the parents. Canonical semantics [CPE #202](https://github.com/mark-e-deyoung/concept-policy-engine-private/issues/202), independent blind protocol [CPE #229](https://github.com/mark-e-deyoung/concept-policy-engine-private/issues/229), public Git storage [CPE #348](https://github.com/mark-e-deyoung/concept-policy-engine-private/issues/348), product [bitmagnet-tools #18](https://github.com/mark-e-deyoung/bitmagnet-tools/issues/18).

## Real item provenance from first successful source rep

Official Internet Archive Advanced Search/Metadata API admitted **two actual .torrent metainfo files** at [GHA 41194283613](https://github.com/mark-e-deyoung/bitmagnet-concept-explorer/actions/runs/41194283613), with a maximum of 30 source item metadata lookups (14 actually examined) and no content payload or peer traffic:

| Original Archive item ID | Source rights claim only | Actual v1 infohash | Metainfo SHA-256 |
|---|---|---|---|
| `download-8_20231018` | Public Domain Mark 1.0 | `1a05b9a28f273ae025bb696e336fb7fd97ca0fa1` | `5a5f5ba3586af31ee0129f2b6d54fe85e2e932aff1f45ea67c681cf89a845d26` |
| `ace_200911_04` | CC0 1.0 | `6145871d04c1f9ddde4347474f5235e4143d5ab9` | `94dc01c6d6053c986062e8e55afabdb9e8ba992f387bfffdddc6c62efeb6225e` |

These are source assertions and **not an independently accepted legal determination**, nor proof of a particular item's suitability for creative reuse. The original .torrent bytes are preserved only as a 14-day [source-run artifact](https://github.com/mark-e-deyoung/bitmagnet-concept-explorer/actions/runs/41194283613), not permanently published to public CPE.

## Live joint RDTE experiment

- Source: real Archive official API and genuine publisher .torrent files, plus genuine Ubuntu publisher torrents. Compute exact raw `info` SHA-1 and complete metainfo SHA-256; import all through real Bitmagnet v0.10.0/PostgreSQL, no DHT or payload.
- Search lane A: one real native Bitmagnet FTS query for an actually ingested Archive identifier, with **exact infohash** verification through the real Explorer `POST /api/v1/sweeps/execute` implementation.
- Search lane B: a second *real* SearXNG JSON-engine adapter that queries **the official Internet Archive AdvancedSearch API**, configured inside the same ephemeral SearXNG container. Require source URL `https://archive.org/details/<identifier>`; no mocked GraphQL or SearXNG data. Keep independent Wikipedia engine for broad research. A SearXNG observation and a Bitmagnet result derived from the **same Archive record** have **one source ancestry**, not two independent corroborations.
- Research flywheel: use existing `POST /api/v1/sweeps/discovery/preview` to produce a CPE #202-compatible `source_mapping_candidate` for Archive's `Archive BitTorrent` taxonomy. Bind its evidence references to the single original Archive source ancestry and immutable experiment inputs. This is only a bounded research proposal; **no canonical CPE concept, source mapping or fact is promoted**.
- Replay metrics: record item identifier, v1 infohash, original metainfo digest, source-rights string, native/external hit counts, candidate proposal ID, `cpe_promotion=false`, `independent_blind_reference=NOT_RUN`. Do not convert a successful retrieval to validated concept recall. The broader Barsoom/Gutenberg source-candidate replay remains independently evidenced by #19.

## Failure and promotion gates

- If the imported Archive torrent does not show up through native FTS, if SearXNG does not return the **same original Archive ID**, if Archive source assertions are missing, or if CPE proposal authority is not `not_accepted_not_transmitted`, **fail closed** and retain first logs. Do not substitute invented identifiers.
- The current sample is deliberately small and predominantly English; multilingual query planning and an EN/DE/TH *scope* in prior #19 do **not** prove genuine independent non-English origin or non-English external results. Record this as a separate qualification gap.
- The CPE #229 frozen independent blind reference and a later real non-English challenge are required before assessing semantic enrichment or genuine multilingual coverage. CPE #202 controls mapping, rejection or promotion. CPE #348 controls a stable public research pack namespace and long-term provenance custody.
- Never put private CPE graphs, local corpus, account logins, disallowed/mixed media feeds, local infrastructure, secret keys or real payloads onto public GHA. The runner is ephemeral, schedule inactive on an unmerged PR, workflow read-only and all service listeners loopback.
