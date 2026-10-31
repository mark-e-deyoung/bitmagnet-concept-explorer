# CPE Git federation source-hardening negative (research only)

Parent: CPE private #348, source PR #23. This independent proposed source rep fixes two reproduced weaknesses: invalid UTF-8 in a concept definition silently serializes as replacement characters under encoding/json; and a self-consistent package+lock could carry arbitrary repository URI strings, including credential/port/query-bearing or file-local origins. The initial RED transcript was preserved in the owning DLE by SHA-256 e6eaf0a28ef7f2ebe97848284a062fc74119f0cfbbb620058ae79806f953bf7d.

The versioned *v0* repository URI grammar deliberately admits only ordinary canonical https://github.com/owner/repo origins (without credentials, ports, query, fragments or path escapes). Additional Git hosts require a separate compatible version negotiation; this is NOT an authenticated remote Git commit fetch or independent source rights review.

The corrected validator rejects invalid UTF-8 before content digesting, preserves exact native Unicode sequences otherwise, and refuses malformed repository origin before constructing read-model output. Tests are synthetic/public-safe and executable offline. Actual Git commit existence/trust and public/private repo visibility still need separate sealed source verification before package release. No semantic acceptance, private-origin publication, merge or deployment.
