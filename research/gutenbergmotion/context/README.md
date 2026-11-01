# C1 — original-source bounded windows (research only)

Consumes the **same** source SHA-pinned Morris Book text and the existing Go miner's
candidate JSON. Validates source commit, blob SHA, full file SHA, source line,
candidate evidence prefix and explicitly unmeasured status before publishing
up to two neighboring nonblank original text lines per side.

```sh
go run ./research/gutenbergmotion/context -source /path/12926.txt -candidates motion-candidates.json > context.json
```

This context is **not** a verified historical figure, notation transcription,
body joint measurement, physical contact or new accepted CPE semantic fact.
Numbered and ambiguous actor references are review cues, not independently
accepted actor tracking. No concept/rights promotion or protected character
processing is performed. The first independent human annotation of whole
figures is still pending.


### Initial context-screening correction

The 64 original lexical occurrences are **not 64 distinct dances**. Multiple
source-window hits occur in the same paragraph and share original lines.
The revised original-source tool deterministically coalesces overlapping
evidence windows into `review_groups` and preserves all original 64 candidate
IDs. These groups are solely analyst **review units**; they do not assert
independent figures, repetitions or reconstructible movement. Original-source
CI is required to verify the observed group count and to reject malformed
intervals, order-dependent group identifiers and any score/physical promotion.

### Bounded first-pass source-checked complete eight-bar figure

The source-owning Go runner now supports:

```sh
go run ./research/gutenbergmotion/context -cross-over-first-pass -source 12926.txt > cross-over-first-pass.json
```

On the exact full pinned book, this yields the **Cross-over** figure, original
lines 1433–1467, written eight bars, seven source-anchored phrase/rule/cue
witnesses: partner crossing (bars 1–2), right-shoulder rule, inward turn
(bars 3–4), recross (bars 5–6), turn to Front (bars 7–8), and jump cue
(bars 4/8). Every phrase is checked against real original-line text before
output, and adverse edited spans, headings or editions fail closed.

**Critical hold:** `TextStatus=SOURCE_SPANS_CHECKED_FIRST_PASS_NOT_INDEPENDENT_GOLD`;
diagrams/notations are not digitally qualified, no music-score source bytes
have been admitted, and an independent second reviewer has not completed
the gold check. The first-pass record is a source-grounded *textual figure*
not validated 3D trajectories, contacts, measured footwork or accepted CPE
semantics. Do not count it as blind heldout evidence.
