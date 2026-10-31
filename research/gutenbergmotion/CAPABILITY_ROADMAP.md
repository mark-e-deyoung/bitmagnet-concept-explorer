# Public research roadmap: music-aware choreography and motion (2026-10-31)

**Status:** PROPOSED / RESEARCH ONLY. No accepted CPE concept/fact promotion, source admission, model training, merge, release or deployment is implied by this roadmap. Public original/synthetic examples only. Source/evidence ownership remains in its existing Git/DLE workcells. Discover → Observe → Plan → Apply → Verify, retaining failures and exact revision/runner receipts.

## Product scope and critical-path rule

The independently owned *private original-character walking MVP* remains P0. Its authentic source/rig, realistic human full-body biomechanics and behavior, trusted private execution, independent video assessment and product acceptance have distinct owners and unresolved gates. **The public historical/music/choreography research described below is not a prerequisite for that MVP.** Existing public synthetic full-body proxy smoke tests do not prove natural anatomy or acceptance. Small reusable clock/foot-phase/rig quality primitives may be adopted only if the product owner finds they materially improve the walking MVP. Do not silently expand MVP acceptance.

## Candidate capabilities, incremental gates

| ID | Increment | Proposed capability/function | Earliest acceptance / existing reuse |
|---|---|---|---|
| C0 | Public research baseline, already proven technically | Source-pinned Go text motion-candidate extraction | Existing independent public draft #22: 159,914-byte original historical book, 64 lexical candidates, original pinned-source native public CI success; semantic reliability UNKNOWN |
| C1 | Research v0.1, P1 | Edition- and source-span-grounded *complete figures*, multiline instruction coreference, roles and hard negatives | Original source+edition/page SHA, one verified 8–16-bar figure with actor/step/timing micro-gold |
| C2 | Research v0.1, P1 | Source-neutral typed choreography score and exact rational bar/beat + time-warp clocks | Reuse existing upstream notation/timeline research and public music MIDI/MusicXML producer; strict actor IDs, source evidence and UNKNOWN |
| C3 | Research v0.2, P1/P2 | Source adapters for Arbeau prose, Playford group dance, Feuillet diagrams, future Tomlinson/Caroso | One human-reviewed complete historical figure per source family; graphical image-to-symbol OCR/vision UNQUALIFIED until checked |
| C4 | Research v0.2, P2 | Culture-native rhythm/step lexicons and CPE research proposals, multilingual source crosswalk | Terms like `compás`, `tāla`, `bol`, `adavu`, `จังหวะ`, `장단` kept native, not silently declared interchangeable; external CPE accepts only through its separate owner |
| C5 | Research v0.2, P2 | Editable music-score ↔ recorded audio ↔ choreographic phase mapping | Reuse public Audio Foundry #9 MIDI 480-PPQ, MusicXML4 and 48 kHz WAV/FFmpeg, exact tempo/repeat/pickup/swing/polymeter and source clock tests |
| C6 | Research v0.3, P2 | Source-score-to-3D synthetic duet/group animation and independent quality assessment | Reuse public Motion #19, Walking #84, VisualQA #150, ROM #170; fail closed on unsupported joint/contact claims and unchanged source evidence |
| C7 | Research v0.3, P2 | Immutable choreography research pack, portable read-only Explorer/API/CLI and consumer input | Reuse existing Git-backed CPE package/lock/OCI/readmodel contracts; two independent same-digest runs, no normative CPE change |
| C8 | Optional P3, defer | Beat-aware learned latent motion retrieval/generation | Compare Bailando/EDGE/Lodge/UniMuMo/MotionBeat only after C1–C7 deliver independently verified score-to-motion evidence; embeddings are NOT semantic facts |
| C9 | Optional P3, defer | Deployment via existing packaging/Foundry lifecycle | Existing owner-approved Foundry F0–F5, immutable OCI+pack digest, no-op same revision, rollback and recover; NO install or registry publication now |

## Preserve separate evidence/time planes

- **Musical source score:** score work/edition, measures, rests, beats, phrases, meter, repeats and native rhythm descriptors. Compatible MusicXML/MEI adapters rather than inventing music notation.
- **Choreographic score:** source-attested figures, role/formation, movement/gesture/turn, intended contact and step support, absolute/relative orientation and ambiguous interpretations.
- **Performance signal:** audio samples and independently observed video frames/PTS, actual performer timing, lead/lag, rubato and recording-specific warp.
- **Kinematic motion:** observed, estimated, authored synthetic or generated 3D joint/root trajectories, biomechanical constraints only when independently qualified, source/rig coordinates/units and uncertainty.

Explicit versioned event IDs and many-to-many time maps bind these planes. Do not equate note beat with footfall, source intention with measured physical motion, image quality with human motion quality, or learned latent similarity with CPE concept acceptance.

## Three-audience UI/UX/DX completion contract (required across C0–C9 and independent MVP)

**All three audiences are first-class consumers**: human creators/reviewers, deterministic automations/operators, and autonomous agents. Each must have UI (the interface by which they interact), UX (an understandable, observable, recoverable task flow), and DX (developer/operator integration and diagnosis affordances). A visual web UI alone is not human acceptance; a JSON API alone is not automation acceptance; tool names without safe plans/typed receipts are not agent acceptance.

| Audience | UI / interaction surfaces | UX / complete task journey | DX / reproducible operations |
|---|---|---|---|
| **Humans** | Accessible responsive GUI, keyboard/assistive access, source page + score + music beat + choreographic actors + clean video, inspect/edit/export/undo controls | Discover source/provenance; inspect original vs interpreted movements; preview synchronized audio/motion; validate; see error/UNKNOWN and exact changes; reverse an edit | Save/load portable versioned project, reproducible inputs and output digest, readable errors, independent file export and repeatable local examples |
| **Automation** | Stable versioned Go CLI and HTTP/JSON endpoints plus machine-readable capability/status/read model | Discover inputs; dry-run plan with estimated cost/rights; submit idempotent execution; poll/stream status; cancel/retry; receive exact verification and artifacts | Deterministic schema/error codes/exit codes, source revision and content ID, timeouts/budgets, concurrency revision guard, integration/test fixtures, no secrets or hidden state |
| **Agents** | Discoverable typed tools/capabilities/schema docs and explicit observable resources; action names and parameter/return contracts | Discover→Observe→Plan→Apply→Verify in bounded scopes, understand uncertainty and source grants; request safe retries; never silently override human/source/release holds | Machine-readable dependency/precondition/rights limits, diff and dry run, immutable receipt and next executable DLE baton, replay/rollback/negative fixtures and distinct FAIL/UNKNOWN/PASS |

**Shared semantic-core invariant:** GUI, CLI, HTTP and agent tools must call the same validated operations and return the same source/plan/score/scene identities, statuses, rights dispositions and artifact hashes. Implement ports/adapters rather than competing controllers or distinct mutable truth. Do not allow one audience to bypass policy/source gates by choosing a different interface.

**Acceptance for EACH capability C0–C9 before advancing status:**
1. Capability inventory links feature IDs to every exposed action and an accessible human journey, CLI/API contract and agent discovery/tool contract; explicitly mark NOT_APPLICABLE with rationale if a surface cannot legitimately do an operation.
2. At least one real-source or original-synthetic positive **full flow** through EACH audience on the same frozen source/package SHA, plus human inspect/modify/reverse where capability is editable; test semantic output parity.
3. Test errors and hard negatives across all surfaces: missing/forged source, rights denial, invalid time/actor/reference, stale revision, concurrent update, cancellation, timeout, out-of-budget execution and scientific `UNKNOWN`/FAIL. Raw errors must not silently become success or a new concept.
4. UI/UX tests include keyboard, screen-reader/accessible labels, focus, responsive viewport, visible source/notation provenance, score/timeline zoom, video with captions/descriptions where needed, explicit save/undo feedback; no claim of complete real iPhone/VoiceOver acceptance from public browser CI.
5. DX tests include OpenAPI/schema or equivalent discoverability, `--help`/capability inquiry, JSON/exit-code parity, idempotent run key, dry-run plan, content SHA, cancellation/retry and rollback to previous read-only package revision; use $0 original-only GHA for hosted tests.
6. Evidence matrix records per-audience/per-operation `PASS`, `FAIL`, `UNKNOWN`, `NOT_RUN`, authority and exact test/run/source SHA. **No averaging:** a green agent smoke cannot conceal a broken human workflow or an unexercised automation rollback.

**MVP boundary:** the independent original-character walking MVP already requires human/automation/agent parity for scene edit/render/inspect/validate; these tests belong on its P0 execution ladder. Music-score UI, dance-specific tabs and cross-cultural choreography authoring are nonblocking P1/P2 capabilities and must not be retroactively made P0 product acceptance criteria. The same three-audience requirements apply when those increments are activated.

## Parallel vs sequential and release discipline

**Parallel READY research, never private product gating:** (A) improve source text phrase+role extraction and independent human reference; (B) source/edition/page-only Arbeau/Playford and Feuillet musical notation discovery; (C) Go original-synthetic 8–16-bar dual-actor score/rational clock contract tests; (D) cross-language semantic hard negatives + independent blind evaluation plan; (E) Audio Foundry original synthetic MusicXML/MIDI/WAV score alignment. Separate ownership and source branch for each, all on standard free public GHA with reviewed original public source. Every source route must be sequential *exact bytes+edition -> transcription/role/score validation -> immutable score manifest -> consumer fanout*.

**Adversarial qualification:** wrong work/edition/page/hash, wrapped text and actor-vs-orientation confusables, invented contact/joint coordinates, ungrounded rhythm translation, performer/song/choreography data leakage, note-repeats/pickups/odd meter/tempo warp, false self-oracle animations, inconsistent rendered videos, Git head concurrency, cross-visibility CPE dependencies and unpinned OCI. No duplicate candidate inserts on replay; changed source produces a superseding immutable revision; rollback changes only active read-only package pointer and does not erase authority history.

**Economics and custody:** prefer Go stdlib, existing original open-source parsers when needed, no second DAW/editor, generic ontology/control-plane service, crawler or new scheduler. Measure *correct complete figures per human review effort*, score-to-performance alignment, wrong-role/contact claims and independent physical/visual defects—not merely hit counts. Small source/metadata/evidence in Git and owning DLE. Large permitted binary deliverables in appropriate controlled Drive storage with SHA/locator; no private subject assets, private repository internals or local infrastructure details in public code or public runners.

**Next baton:** complete C1 and C2 source/synthetic control experiments while independently finishing C5 one original synthetic 8–16-bar score/audio fixture. Freeze source-validated score reference before any CPE discovery comparison or downstream model training. Release and production deployment are not authorized.
