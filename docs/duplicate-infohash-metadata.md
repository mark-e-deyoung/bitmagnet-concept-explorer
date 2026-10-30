# Duplicate infohash metadata

A Bitmagnet query can return the same infohash on multiple tracks or terms. The executor merges those rows into one result and retains the sorted union of `tracks` and `matched_queries` as retrieval provenance.

Metadata uses a conservative field-by-field merge:

- Missing values do not erase a known value.
- Equal known values remain known. Language IDs are compared as sorted unique sets.
- When two known values disagree, that field becomes unknown: strings become empty, numeric and adult metadata pointers become `null`, and languages become an empty list. The field name is included in `conflicting_metadata`.
- Once a field conflicts, later responses cannot restore a value. Completion order therefore cannot choose which conflict value is exposed or let it qualify a metadata filter.
- Filters and groups operate on the merged bounded results. Unknown values do not match concrete filters and are grouped as `unknown` where applicable.

This is retrieval metadata reconciliation only. It makes no claim about content, age, safety, source trust, canonical concepts, or policy. No extra Bitmagnet request is made to resolve a conflict.
