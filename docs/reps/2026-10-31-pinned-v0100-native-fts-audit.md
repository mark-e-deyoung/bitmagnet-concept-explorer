# Pinned upstream Bitmagnet v0.10.0 FTS boundary — source-only correction

Authority: private product DLE `mark-e-deyoung/bitmagnet-tools#18`, older exact-source review baton comment 6537334806. This is a technical retrieval review, **not** CPE owner acceptance or permission to widen the lexical query contract.

## DISCOVER / first source evidence

- Upstream **exact** Bitmagnet release tag `v0.10.0`, resolved commit `4d4042acdddfa1af05bfaf092a68c97b1e3698e2`.
- `internal/database/fts/tsquery.go` blob `a50b2ab4adb662779799b6928be2e5f3e84731d0`: `readQueryToken` recognizes `& | . ! * ( )` and quoted tokens; `appQueryTokensToTsquery` inserts `&` by default between unquoted phrase tokens, while a quoted token joins lexemes with `<->`.
- Upstream `internal/database/fts/tsquery_test.go` blob `25294945f773a58d4c229888055a0fc475e4c290` explicitly expects `foo bar -> foo & bar` and quoted `"make me a " -> make <-> me <-> a`.
- Upstream `internal/lexer/lexer.go` blob `fe079a95dc5063582694154cb75ffd013aac57fb`: `IsWordChar` admits letters and digits, not combining marks. The tokenizer transliterates non-ASCII codepoints. Phrase and native-language equivalence remain unqualified without exact runtime differential tests.
- PR #2's `validateBitmagnetSafeLiteral` rejects operators, quotes, punctuation, controls, repeated ASCII spaces, and leading/trailing whitespace. However `science fiction` still passes and goes to native `queryString` unquoted, so the upstream parser treats it as an **AND retrieval query**, not an exact phrase. It is safe from the listed operator injection but is **not literal phrase preservation**.

## PLAN / ACT — smallest safe correction

Do not silently quote or escape CPE-generated terms, broaden admission, or alter direct user-authored FTS. Those choices need CPE owner/retrieval acceptance. Instead, append an explicit warning for retained multiword `digest-pinned-preview:`/`pinned-preview:` terms to both query plans and sweep plans; no new network calls or authority paths. Source-only synthetic regression verifies one warning, no native-query rewrites, and unchanged read-only/UNKNOWN boundaries.

The warning is not a claim of comprehensive language-sense equivalence; Unicode transliteration/tokenization and retrieval precision still need native exact-version qualification.

## VERIFY / gates

Focused: `go test -count=1 -run 'TestPinnedFTSMultiword' ./...` plus repository's offline exact-head public Actions qualification (`go test -count=1 ./...`, `go test -count=1 -race ./...`, `go vet ./...`, `gofmt -l .`, native amd64/arm64 builds and existing `node web/query_state.test.mjs`). Record each actual result in DLE, not assertions from source review.

CPE owners must determine whether multiword CPE lexicalizations should remain broad retrieval candidates, be exact-phrase quoted after tokenizer qualification, or be explicitly omitted. No semantic promotion, publication, Foundry, canary, deployment, or merge is authorized by this correction. Evidence in Git; baton/decision in private DLE; no large binary or Drive artifact.
