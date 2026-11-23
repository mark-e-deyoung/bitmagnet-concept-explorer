#!/usr/bin/env python3
"""Independent, read-only research snapshot consumer (CPE #202, R35).

This is NOT a reconciler, source authority, knowledge release issuer, policy
enforcement point, blob fetcher, or language-equivalence classifier. It reads an
existing pinned Go-produced v0 projection with a required external expected
source-lock digest and answers a typed concept/lexicalization query offline.
No third-party libraries or network calls.
"""
from __future__ import annotations

import argparse
import json
from pathlib import Path
import re
import sys
from typing import Any

MAX_BYTES = 2 << 20
ID = re.compile(r"[A-Za-z0-9][A-Za-z0-9:._-]{0,159}\Z", re.ASCII)
SHA256 = re.compile(r"[0-9a-f]{64}\Z", re.ASCII)


class Rejected(ValueError):
    pass


def pairs_unique(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    result: dict[str, Any] = {}
    for key, value in pairs:
        if key in result:
            raise Rejected("DUPLICATE_JSON_KEY")
        result[key] = value
    return result


def _constant_rejected(_value: str) -> None:
    raise Rejected("INVALID_JSON_CONSTANT")


def _string(s: Any, label: str) -> str:
    if not isinstance(s, str) or not s or any(0xD800 <= ord(ch) <= 0xDFFF for ch in s):
        raise Rejected("INVALID_" + label)
    return s


def _typed_id(s: Any) -> str:
    s = _string(s, "CONCEPT_ID")
    if ID.fullmatch(s) is None:
        raise Rejected("INVALID_CONCEPT_ID")
    return s


def _exact_fields(obj: Any, required: set[str], optional: set[str] = frozenset()) -> dict[str, Any]:
    if not isinstance(obj, dict) or not required.issubset(obj) or set(obj) - required - optional:
        raise Rejected("UNSUPPORTED_SNAPSHOT_SCHEMA")
    return obj


def read_projection(path: Path, expected_sha: str) -> dict[str, Any]:
    if SHA256.fullmatch(expected_sha) is None:
        raise Rejected("INVALID_EXPECTED_LOCK")
    try:
        with path.open("rb") as stream:
            raw = stream.read(MAX_BYTES + 1)
    except OSError as exc:
        raise Rejected("UNREADABLE_SNAPSHOT") from exc
    if len(raw) > MAX_BYTES:
        raise Rejected("OVERSIZED_SNAPSHOT")
    try:
        document = json.loads(
            raw.decode("utf-8", "strict"),
            object_pairs_hook=pairs_unique,
            parse_constant=_constant_rejected,
        )
    except (UnicodeError, json.JSONDecodeError) as exc:
        raise Rejected("MALFORMED_SNAPSHOT") from exc
    doc = _exact_fields(
        document, {"schema_version", "authority", "lock_sha256", "concepts"}
    )
    if doc["schema_version"] != "cpe-readmodel-v0" or doc["authority"] != "research_only":
        raise Rejected("UNADMITTED_SNAPSHOT_AUTHORITY")
    sha = doc["lock_sha256"]
    if not isinstance(sha, str) or SHA256.fullmatch(sha) is None or sha != expected_sha:
        raise Rejected("SOURCE_LOCK_MISMATCH")
    concepts = doc["concepts"]
    if not isinstance(concepts, list):
        raise Rejected("INVALID_CONCEPT_COLLECTION")
    indexed: dict[str, dict[str, Any]] = {}
    for item in concepts:
        c = _exact_fields(item, {"id", "definition"}, {"lexicalizations", "relations"})
        cid = _typed_id(c["id"])
        _string(c["definition"], "DEFINITION")
        if cid in indexed:
            raise Rejected("DUPLICATE_CONCEPT_ID")
        lexical = c.get("lexicalizations", [])
        relations = c.get("relations", [])
        if not isinstance(lexical, list) or not isinstance(relations, list):
            raise Rejected("INVALID_CONCEPT_DETAILS")
        seen_lexical: set[tuple[str, str]] = set()
        for value in lexical:
            l = _exact_fields(value, {"language", "term"})
            pair = (_string(l["language"], "LANGUAGE"), _string(l["term"], "TERM"))
            if pair in seen_lexical:
                raise Rejected("DUPLICATE_LEXICAL_EVIDENCE")
            seen_lexical.add(pair)
        seen_relations: set[tuple[str, str]] = set()
        for value in relations:
            rel = _exact_fields(value, {"predicate", "target_id"})
            pair = (_typed_id(rel["predicate"]), _typed_id(rel["target_id"]))
            if pair in seen_relations:
                raise Rejected("DUPLICATE_CONCEPT_RELATION")
            seen_relations.add(pair)
        indexed[cid] = c
    for item in indexed.values():
        for rel in item.get("relations", []):
            if rel["target_id"] not in indexed:
                raise Rejected("UNRESOLVED_CONCEPT_RELATION")
    return {"source_lock_sha256": sha, "concepts": indexed}


def query(projection: dict[str, Any], concept_id: str, language: str | None) -> dict[str, Any]:
    cid = _typed_id(concept_id)
    concept = projection["concepts"].get(cid)
    if concept is None:
        return {"status": "UNKNOWN_CONCEPT", "concept_id": cid,
                "authority": "research_only", "source_lock_sha256": projection["source_lock_sha256"]}
    if language is not None:
        _string(language, "LANGUAGE")
    lex = concept.get("lexicalizations", [])
    if language is not None:
        lex = [value for value in lex if value["language"] == language]
    return {
        "status": "OBSERVED" if lex or language is None else "NO_LEXICALIZATION",
        "concept_id": cid,
        "definition": concept["definition"],
        "lexicalizations": lex,
        "relations": concept.get("relations", []),
        "authority": "research_only",
        "source_lock_sha256": projection["source_lock_sha256"],
    }


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--snapshot", required=True, type=Path)
    parser.add_argument("--expected-lock-sha256", required=True)
    parser.add_argument("--concept-id", required=True)
    parser.add_argument("--language", default=None)
    args = parser.parse_args(argv)
    try:
        projection = read_projection(args.snapshot, args.expected_lock_sha256)
        result = query(projection, args.concept_id, args.language)
    except Rejected as exc:
        print(str(exc), file=sys.stderr)
        return 3
    print(json.dumps(result, sort_keys=True, ensure_ascii=False, separators=(",", ":")))
    return 0 if result["status"] != "UNKNOWN_CONCEPT" else 4


if __name__ == "__main__":
    raise SystemExit(main())
