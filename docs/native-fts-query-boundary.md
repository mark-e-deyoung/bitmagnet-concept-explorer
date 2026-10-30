# Native FTS query boundary

User-authored direct queries retain native Bitmagnet full-text syntax and are passed through unchanged. A lexicalization carrying pinned CPE-preview provenance is a candidate literal, not a request to compose FTS operators. It is admitted only when it contains Unicode letters, marks, numbers, and single ASCII spaces; controls, punctuation, quotes, and parser operators are rejected during planning. The same boundary applies to graph-derived terms in multilingual sweep plans.

The planner does not quote, escape, tokenize, or rewrite a rejected CPE-generated term. Bitmagnet's exact literal-phrase encoding has not been qualified here, and silent rewriting could change retrieval. The legacy historical lower-preview continues to show its non-executable quoted representation separately. Manual unverified expansions remain user-authored input and retain existing native query behavior.

This is a retrieval syntax boundary, not a CPE semantic or policy change. It requires review against the exact pinned Bitmagnet tokenizer before any broader literal-query contract is claimed.
