# Public CPE source-grounded motion-text mining, rep 001

**Purpose.** Mine real motion instructions from the public historical dance manual *The Morris Book, Part 1* (Project Gutenberg ebook #12926). This is a *research candidate* source for public CPE/SCP crosswalk, not an accepted CPE ontology change or 3D motion-capture dataset.

**Pinned primary source and mirror.** Original source: https://www.gutenberg.org/ebooks/12926. Read-only GITenberg mirror: https://github.com/GITenberg/The-Morris-Book-Part-1--13-A-History-of-Morris-Dancing-With-a-Description-of-Eleven-Dances-as__12926. Pin Git commit `1d5ca33372491509805a64cd0425a5164d3108fa`, file `12926.txt`, blob `08984be691ebd69f92a7a271217612f0056cea37`. The GHA workflow independently verifies its source Git object hash before parsing. Do not scrape Gutenberg.org from runners; use its authorized mirrors and metadata feeds.

**Command:** `go run ./research/gutenbergmotion -input /path/to/12926.txt > motion-candidates.json`

**Bounded extraction:** A line must contain a recognizable movement term, actor term and musical time reference. Every result is tied to an original source line plus content SHA-256; actor/timing/motion words remain source vocabulary; any missing measurement stays `UNMEASURED_TEXT_INSTRUCTION`. This v0 scanner is deliberately conservative, may miss motions spanning multiple lines and may propose ambiguous phrases. Output labels are hypotheses, not confirmed 3D joints, force/contact, physical timing or semantic truth. Test negatives prevent unsupported associations.

**Next reps (existing owners, no duplicate controllers):**
1. Human-verified source spans, original notation glossary and paragraph/table/diagram segmentation.
2. Parse bars and beats into a rational music-phase timeline, reusing existing SCP MMD #19 notation research; role assignments and figure formation topologies.
3. Emit source-bound CPE research proposals through existing CPE #202 and independently challenge through #229; no direct promotion.
4. Lower qualified choreography constraints into SCP's source-neutral motion representation; test with original synthetic avatar rigs and physical/visual QA.
5. Add Feuillet, Arbeau and Playford sources only after recording exact editions, source files, notation and independence; no fabricated transcriptions.

**Custody:** Public source/Go/fixture/metadata/compact evidence in Git and cross-linked DLE. Public GHA artifact is short-lived, reproducible and digest-indexed; only larger retained deliverables should use appropriate Drive storage. No public redistribution of private CPE authority or private real-person sources. Draft PR only; no merges or releases.
