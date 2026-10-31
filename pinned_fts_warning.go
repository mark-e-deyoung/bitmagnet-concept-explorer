package main

// This describes a proven upstream retrieval boundary, not CPE adjudication.
// Bitmagnet v0.10.0 AppQueryToTsquery uses implicit AND between unquoted words.
const pinnedMultiwordFTSWarning = "Pinned Bitmagnet v0.10.0 parses unquoted multiword CPE preview terms with AND, not phrase adjacency; results are broad retrieval hypotheses, not exact-phrase or concept evidence."
