# Accessibility, platform behavior, and recovery contract

BCE is a browser application over the same read-only service used by the CLI and
automation. Its human interface must remain usable with a keyboard, speech input,
screen reader, magnification, touch, and user-selected display settings. Browser
semantics, native operating-system behavior, and actual assistive-technology (AT)
usability are separate qualification surfaces. Passing one does not establish the
others.

This contract applies to the public application in `web/index.html`, its
synthetic browser fixtures, and client-facing documentation. It does not expand
provider consent, metadata access, CPE authority, or deployment authorization.

## Audiences and successful outcomes

| Audience | Required outcome | Interface requirement |
| --- | --- | --- |
| Occasional human researcher | Enter a phrase, inspect direct and expanded evidence, understand what is uncertain | A prominent labeled search, optional refinements, explicit results state, understandable failure and recovery instructions |
| Frequent analyst | Compare metadata, inspect provenance, follow a selected source candidate, inspect multilingual concepts | Stable keyboard focus, full labels beside graph alternatives, source identity and uncertainty preserved, deliberate opt-in before expansions or external requests |
| Human using AT or alternative input | Complete the same tasks without needing to see, point at, or drag the graph | Native controls, programmatic names and state, logical landmarks, visible focus, accessible descriptions and errors, no essential hover-only information |
| Operator | Distinguish configuration, dependency availability, and acceptance | Capability checks described as configuration; provider failure stays visible; no successful-looking empty output after failure |
| CLI user or automation author | Obtain bounded, structured results and act on failure without parsing prose | Documented request/response fields, meaningful error/exit behavior, structured partial and failed states, no implicit retry or external lookup |
| Reasoning agent | Preserve attribution and recover within its authority | Read capabilities, retain failures and provenance, distinguish evidence from semantic acceptance, never treat an unavailable source as absence |
| Contributor or reviewer | Reproduce a defect and assess a correction | Public synthetic fixtures, exact revision and environment receipts, focused regressions, documented limitations of automated accessibility checks |

## Platform scope

Native web controls and the browser's accessibility implementation are the
default. Do not emulate an operating system's menus, dialogs, radio groups, or
keyboard shortcuts in application JavaScript. Use custom ARIA widgets only when
the task requires them and their complete interaction contract can be supported.
Do not intercept browser, OS, or AT shortcuts. [S1][S8][S9][S10]

| Client surface | Source-level expectation | Qualification still needed before an AT support claim |
| --- | --- | --- |
| Desktop Chromium, Firefox, WebKit | Native form controls, browser zoom, logical Tab order, visible focus, accessible result/status semantics | A recorded task run on each claimed browser/OS pairing; engine-only CI is narrower evidence |
| Windows browser, including Edge | Keyboard navigation, system fonts, high contrast/forced colors, accessible names and control state | Native Windows with the claimed Edge/Chrome/Firefox version plus Narrator and/or NVDA; Linux Edge cannot establish Windows UIA or AT behavior |
| macOS Safari | Browser-native controls, expected keyboard navigation, system appearance, reduced motion, zoom | Safari on macOS with VoiceOver and the required keyboard-navigation setting documented; Playwright WebKit is not Safari+VoiceOver |
| iOS/iPadOS Safari | Reflow, pinch zoom, readable text, usable touch targets, understandable focus/reading order | Mobile Safari with VoiceOver and touch exploration, text enlargement and software keyboard; a simulator screenshot establishes only its captured visual state |
| Android Chrome | Reflow, usable touch targets, descriptive controls, alternatives to precise gestures | Chrome on a named Android device/emulator with TalkBack and, where claimed, Switch Access; an Android-sized Chromium viewport is not Android AT evidence |
| Linux desktop browser | Standard browser controls and keyboard/display preferences | An actual Orca/browser/desktop pairing before claiming that pairing is supported |

The current public workflows contain Linux browser engines, Edge on Linux,
mobile-sized browser profiles, macOS WebKit, and an iOS Simulator capture.
Inspect workflow files and exact-head receipts for what actually ran. In
particular, do not turn an `openurl` exit code or a nonempty screenshot into an
accessibility or end-to-end task acceptance claim.

The TrueNAS host does not change the browser client's interaction conventions.
Foundry and native TrueNAS lifecycle qualification remain separate from this
web-interface contract.

## Required interaction behavior

### Structure, labels, and language

- The first keyboard entry offers a skip link to a named main landmark. Headings
  and sections describe tasks in reading order; navigation links have existing
  destinations. Do not use `role="application"` for the workbench.
- Every input retains a persistent visible label. Optional fields, constraints,
  and provider-specific formats are stated before submission. Placeholder text
  is supplemental, never the only label.
- A control's accessible name contains its visible action label. Context can be
  appended: for example, **Inspect media sources for Synthetic film**. Repeated
  candidate actions include the source/title needed to distinguish them. Dynamic
  copy feedback must not leave a stale accessible name. [S2]
- Native buttons activate with Enter and Space; native radio groups retain
  browser arrow-key behavior; disclosures retain `<details>/<summary>` behavior.
  Avoid positive `tabindex` and custom single-character shortcuts. [S1]
- Multilingual lexicalizations preserve their full Unicode text, known language
  tags, and source attribution. Use `lang` for known human-language spans and
  direction isolation for source text. Do not infer a language from script alone.

### Graphs remain optional presentations

Every graph action has an equivalent, visible native HTML button with the full
concept label and selected state. A scrollable diagram has a name, keyboard
focus, and instructions. Truncated SVG labels must not truncate the corresponding
control's accessible name. Concept selection may prepare a query, but never
executes it or silently opts in to lexical expansions.

The text detail exposes the selected concept, definition, identifiers,
lexicalizations, relationship predicates and endpoints. Observed-evidence
details expose the matching source field and term and state their research
eligibility. A line's color or geometry is not the sole way to learn a
relationship. When rendering replaces a selected graph node or native selector,
restore focus to the replacement of that same control. Do not move focus merely
because a background catalog or result response arrived. [S1]

There is no need for an application modal to complete these tasks. Future
dialogs require a separate dialog-pattern review, including initial focus,
contained navigation, Escape behavior where appropriate, and focus return.

### Validation, status, and failure recovery

1. On an explicit invalid submission, identify the field and explain the remedy
   in text. Associate it through `aria-describedby` or equivalent supported
   semantics, set `aria-invalid="true"`, and focus the implicated control when
   that is the direct correction path. Preserve any existing help reference.
   Native required/range validation remains available. [S3]
2. Correcting a field removes its stale error state. Validation must not erase
   unrelated input or turn an optional external action on. Cross-field conflicts
   such as genre search combined with title lookup explain which fields to clear.
3. Missing provider consent is a local validation failure. Associate the error
   with the consent control and send no provider request. A checkbox is the user's
   decision, not a suggestion the app may automatically accept.
4. Use stable, concise live status elements for progress, success, partial
   completion, cancellation and errors. Expose status without requiring focus on
   the message. Use `aria-busy` on the content being replaced; keep the status
   announcement outside that busy subtree so progress is not withheld. Avoid
   making every result card or full JSON document a live region. [S4]
5. On success or asynchronous failure, preserve the user's current editing or
   reading focus. The result summary and available recovery action must remain
   discoverable. Move focus only for a deliberate navigation or correction, not
   as a side effect of response timing.
6. Every request terminal path restores its controls: complete, partial, failed,
   invalid response, timeout, cancellation, or supersession. A stale response
   cannot replace the current query's evidence. A new search must not leave a
   superseded sweep's controls disabled.
7. Keep `complete`, `partial`, `failed`, `blocked`, and `canceled` distinguishable
   in text and structured state. Failed or unavailable sources do not establish
   absence. A local preview states exactly what work it executes and whether
   evidence was persisted, submitted, or accepted.

### Display and input preferences

- Preserve browser zoom. Do not use `user-scalable=no`, a restrictive
  `maximum-scale`, or fixed-size containers that clip enlarged text.
- At 320 CSS pixels, ordinary content reflows without page-wide horizontal
  scrolling or loss of task functionality. A genuinely two-dimensional diagram
  can scroll in its own region while its text alternative reflows. This is a
  400%-equivalent reflow check for a 1280 CSS-pixel starting viewport; it is not an
  actual browser-zoom or OS text-size test. [S5]
- At 200% text enlargement, labels, statuses, actions and navigation remain
  readable and operable. Support line-height 1.5, paragraph spacing 2em,
  letter-spacing .12em and word-spacing .16em without clipping or overlap. Test
  those spacing overrides independently of other changes. [S6][S8]
- Keep focused controls visible. Sticky content must not cover a field or its
  focus indicator after scrolling, validation, zoom, or text enlargement. Prefer
  allowing the header to scroll when its wrapped size makes this unreliable. [S7]
- Respect system light/dark appearance by default and allow an optional local
  override. User-selected forced colors must retain text, borders, focus and
  selected-state distinctions. Never rely solely on colored state borders.
- Respect reduced motion for smooth scrolling and decoration. Necessary
  progress remains available as text. Do not animate essential status away.
- Meet WCAG 2.2 target size and spacing requirements; use generous 44–48 CSS-pixel
  touch areas as a product usability target where practical. A CSS pixel is not
  an Apple point or an Android density-independent pixel; check native touch
  behavior on the claimed platform rather than asserting unit equivalence.
  Inline links and other exceptions need their own assessment. [S8][S9][S10][S11]

## Findings that drive this correction

These are source-level observations from the public PR #79 base, not a claim that
an affected user has completed a native AT evaluation.

| Priority | Existing surface | Defect or evidence gap | Required correction and regression |
| --- | --- | --- | --- |
| High | `displayItems()` / `.media-inspect` | Visible “Inspect media sources” differs from its “Inspect catalog matches…” accessible name | Preserve the visible phrase in the accessible name; test a populated movie result through the accessibility tree |
| High | Media lookup and sweep client validation | Status-only errors do not associate the remedy with the relevant control or restore the correction path | Associate error text, expose invalid state, focus missing input/consent, preserve help, and prove no request was sent |
| High | `clearQueryEvidence()` / `sweepAction()` | Superseding a sweep invalidates its token while its finalizer only restores controls for the old token | Restore actionable controls and prove late responses cannot overwrite the current request |
| Medium | `.hash-copy` | Visible text changes on copy success/failure while its accessible name can remain stale | Keep the visible action and accessible name aligned; announce the outcome in a stable status |
| Medium | Source-candidate buttons | Repeated action names omit the candidate's distinguishing context | Append title/source to the accessible action name and preserve the visible action phrase |
| Medium | Cross-field validation and programmatic prefills | Fixing a conflict in another field can leave obsolete `aria-invalid`; programmatic selection does not emit an input event | Revalidate the form without retaining obsolete error associations; clear errors on deliberate prefills |
| High | Media lookup scope replacement | Editing the request or selecting a new observed item while a lookup is pending can present old candidates beside new input | Abort/invalidate the old request and discard its response; preserve the new input and keyboard focus |
| Medium | Result loading and error states | No explicit busy state; transport success can style partial/failed service results as ready | Expose content busy state and explicit semantic result outcomes; retain recovery and field values |
| Medium | Sticky header and dense forms | Fixed scroll padding does not prove focus visibility when content wraps or text grows | Test focus visibility, 320px reflow, 200% text, spacing overrides, and touch hit regions |
| Medium | Native platform claims | Engine profiles and simulator screenshots lack actual screen-reader assertions | Separate automated checks from native interaction and AT receipts |

## Verification and evidence

`tests/ui/accessibility-recovery.spec.mjs` uses only synthetic intercepted API
responses. It exercises keyboard entry, result semantics, labels used by speech
input, invalid-input/consent recovery, graph alternatives, display preferences,
and reflow. Unknown API routes fail the fixture instead of reaching a provider.
Run it in every configured browser family to which the support claim applies.

```sh
npx playwright test tests/ui/accessibility-recovery.spec.mjs
npx playwright test --config playwright.crossbrowser.config.mjs accessibility-recovery.spec.mjs
```

Both crossbrowser and Edge/mobile configurations select this suite. The
inventory validator checks those inclusions as well as executable test markers;
a control mapped only to an unselected test file does not satisfy that check.
Browser executables and the pinned test dependencies must be installed before
either command can produce execution evidence. A syntax check, inventory pass,
or discovered test count is not a browser pass.

Automated assertions inspect browser accessibility semantics and simulated
preferences. They cannot prove announcement timing, pronunciation, virtual-cursor
navigation, rotor behavior, touch exploration, speech recognition accuracy,
comfort, or task completion for people using AT. Axe and Lighthouse supplement
functional tests and human evaluation; neither gives complete WCAG conformance.

For each native AT receipt record: exact source revision, OS/browser/AT versions,
device and input method, settings (keyboard navigation, zoom/text size, contrast,
motion), task and synthetic data, expected result, observed announcement/focus,
pass/failure, and reviewer. Retain failures and describe any workaround. Do not
mark an unrun combination as passed.

Minimum native task script:

1. Enter from the address bar; use the skip link and labeled search. Submit with
   the keyboard or AT action and hear the bounded request and result status.
2. Read a direct/shared result, its metadata and uncertainty; use a distinctly
   named source-inspection action; recover from missing input and consent.
3. Inspect a full-name concept and its relationships using the HTML alternative;
   verify selection and focus; opt in deliberately, then search.
4. Trigger a synthetic source failure, then retry explicitly; verify prior
   evidence is not presented as current and no absence inference is suggested.
5. Repeat at large text/zoom, with the OS contrast/appearance preference and
   reduced motion. On touch platforms verify exploration, activation, scrolling,
   software-keyboard visibility and return to the edited control.

## Primary guidance

The references below guide the web implementation. Native-framework examples
inform platform expectations; they do not require adding UIKit, WinUI or Compose
to this browser application. Guidance reviewed 2026-11-28.

- **S1:** [WAI-ARIA APG: Developing a Keyboard Interface](https://www.w3.org/WAI/ARIA/apg/practices/keyboard-interface/).
- **S2:** [WCAG 2.2: Label in Name](https://www.w3.org/WAI/WCAG22/Understanding/label-in-name.html).
- **S3:** [WCAG 2.2: Error Identification](https://www.w3.org/WAI/WCAG22/Understanding/error-identification.html).
- **S4:** [WCAG 2.2: Status Messages](https://www.w3.org/WAI/WCAG22/Understanding/status-messages.html).
- **S5:** [WCAG 2.2: Reflow](https://www.w3.org/WAI/WCAG22/Understanding/reflow.html).
- **S6:** [WCAG 2.2: Text Spacing](https://www.w3.org/WAI/WCAG22/Understanding/text-spacing.html).
- **S7:** [WCAG 2.2: Focus Not Obscured (Minimum)](https://www.w3.org/WAI/WCAG22/Understanding/focus-not-obscured-minimum.html).
- **S8:** [Apple Human Interface Guidelines: Accessibility](https://developer.apple.com/design/human-interface-guidelines/accessibility).
- **S9:** [Microsoft: Designing inclusive software for Windows](https://learn.microsoft.com/en-us/windows/apps/design/accessibility/designing-inclusive-software).
- **S10:** [Android: Principles for improving app accessibility](https://developer.android.com/guide/topics/ui/accessibility/principles).
- **S11:** [WCAG 2.2: Target Size (Minimum)](https://www.w3.org/WAI/WCAG22/Understanding/target-size-minimum.html).
