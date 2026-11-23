#!/usr/bin/env python3
"""R36 research-only, privacy-minimized concept-discovery proposal producer.

Consumes the independent R35 read-only projection; DOES NOT write or accept
concepts, infer equivalence, authorize rights, or contact a provider/network.
Strictly ORIGINAL public synthetic fixtures in this qualifying rep.
"""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import re
import sys
from typing import Any

import consumer_r35 as consumer

VERSION = "cpe-discovery-proposal-research-v0"
LOCALE = re.compile(r"[a-z]{2,3}(?:-[A-Za-z0-9]{2,8}){0,3}\Z", re.ASCII)
SOURCE_ID = re.compile(r"[A-Za-z0-9][A-Za-z0-9:._-]{0,159}\Z", re.ASCII)
HEX40 = re.compile(r"[0-9a-f]{40}\Z", re.ASCII)
HEX64 = re.compile(r"[0-9a-f]{64}\Z", re.ASCII)


class ProposalRejected(ValueError):
    pass


def _hex(text: Any, regexp: re.Pattern[str], reason: str) -> str:
    if not isinstance(text, str) or regexp.fullmatch(text) is None:
        raise ProposalRejected(reason)
    return text


def _ascii_id(text: Any, reason: str) -> str:
    if not isinstance(text, str) or SOURCE_ID.fullmatch(text) is None:
        raise ProposalRejected(reason)
    return text


def _native_term(value: Any) -> str:
    if not isinstance(value, str) or not value or len(value) > 96:
        raise ProposalRejected("INVALID_NATIVE_TERM")
    if any(ord(ch) < 32 or 0xD800 <= ord(ch) <= 0xDFFF for ch in value):
        raise ProposalRejected("INVALID_NATIVE_TERM")
    # This only rejects obvious locators, not sensitive content. Never supply
    # arbitrary private terms to this public-synthetic research adapter.
    if "://" in value or "/" in value or "\\" in value:
        raise ProposalRejected("POTENTIAL_RAW_LOCATOR")
    return value


def build_proposal(
    projection: dict[str, Any],
    *,
    target_concept_id: str,
    language: str,
    observation_type: str,
    candidate_term: str | None,
    source_system: str,
    evidence_sha256: str,
    producer_revision: str,
    snapshot_sha256: str,
    privacy_class: str = "synthetic_public_only",
) -> dict[str, Any] | None:
    """Deterministic bounded candidate envelope, not adjudicated ontology data.

    If a consumer miss occurs even though a lexicalization exists, emit a
    COVERAGE_GAP rather than invent a new mapping. A native source term that
    is not already represented becomes a LEXICALIZATION_CANDIDATE only when
    the target concept is already present; its equivalence remains UNKNOWN.
    """
    cid = consumer._typed_id(target_concept_id)
    if not isinstance(language, str) or LOCALE.fullmatch(language) is None:
        raise ProposalRejected("INVALID_LANGUAGE")
    if observation_type not in {"native_source_term", "consumer_lookup_miss"}:
        raise ProposalRejected("INVALID_OBSERVATION_TYPE")
    if observation_type == "native_source_term":
        term = _native_term(candidate_term)
    elif candidate_term is not None:
        raise ProposalRejected("UNEXPECTED_NATIVE_TERM")
    else:
        term = None
    _ascii_id(source_system, "INVALID_SOURCE_SYSTEM")
    evidence = _hex(evidence_sha256, HEX64, "INVALID_EVIDENCE_DIGEST")
    revision = _hex(producer_revision, HEX40, "INVALID_PRODUCER_REVISION")
    snap = _hex(snapshot_sha256, HEX64, "INVALID_SNAPSHOT_DIGEST")
    if privacy_class != "synthetic_public_only":
        raise ProposalRejected("UNQUALIFIED_PRIVACY_CLASS")
    source_lock = _hex(projection.get("source_lock_sha256"), HEX64, "INVALID_SOURCE_LOCK")
    observed = consumer.query(projection, cid, language)
    kind = "coverage_gap"
    why = "CONSUMER_CAPABILITY_OR_PROJECTION_UNKNOWN"
    relationship = "unmapped"
    if observed["status"] == "UNKNOWN_CONCEPT":
        why = "UNKNOWN_CONCEPT_NOT_PROOF_OF_NOVELTY"
    elif observation_type == "native_source_term":
        # Existing same-language exact term is not a new proposal. No inference
        # from lexical similarity/translation is permitted.
        if any(x["term"] == term for x in observed["lexicalizations"]):
            return None
        kind = "lexicalization_candidate"
        why = "NATIVE_SOURCE_TERM_NOT_IN_PINNED_LEXICAL_VIEW"
    else:
        # Miss on an existing concept is consumer retrieval evidence, not
        # reason to extend the global ontology.
        why = "KNOWN_CONCEPT_RETRIEVAL_MISS_NOT_ONTOLOGY_GAP"
    body = {
        "schema_version": VERSION,
        "authority": "RESEARCH_PROPOSAL_ONLY",
        "adjudication": "UNADJUDICATED",
        "proposal_kind": kind,
        "producer": {"component": "bce.independent_python_consumer.r35",
                     "exact_revision": revision},
        "source_projection": {
            "authority": "research_only",
            "source_lock_sha256": source_lock,
            "snapshot_bytes_sha256": snap,
        },
        "observation": {
            "target_concept_id": cid,
            "language": language,
            "observation_type": observation_type,
            "native_term": term,
            "consumer_status": observed["status"],
            "consumer_impact": "RETRIEVAL_MISS_OR_MAPPING_GAP",
            "source_system": source_system,
        },
        "evidence": {
            "provenance_digest": evidence,
            "source_ancestry": [evidence],
            "support_count": 1,
            "independent_corroboration": "NOT_ESTABLISHED",
            "confidence": None,
            "falsifier_status": "NOT_YET_CHALLENGED",
        },
        "mapping": {
            "candidate_relation": relationship,
            "equivalence_status": "UNKNOWN",
            "classification": why,
            "candidate_not_accepted": True,
        },
        "privacy": {
            "class": privacy_class,
            "raw_private_content_included": False,
            "sanitization_independently_qualified": False,
        },
    }
    canonical = json.dumps(body, ensure_ascii=False, sort_keys=True,
                           separators=(",", ":"), allow_nan=False).encode("utf-8")
    digest = hashlib.sha256(VERSION.encode("ascii") + b"\x00" + canonical).hexdigest()
    return {"proposal_id": "proposal:sha256:" + digest, **body}


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--snapshot", type=Path, required=True)
    parser.add_argument("--expected-lock-sha256", required=True)
    parser.add_argument("--expected-snapshot-sha256", required=True)
    parser.add_argument("--concept-id", required=True)
    parser.add_argument("--language", required=True)
    parser.add_argument("--observation-type", required=True,
                        choices=("native_source_term", "consumer_lookup_miss"))
    parser.add_argument("--term", default=None)
    parser.add_argument("--source-system", required=True)
    parser.add_argument("--evidence-sha256", required=True)
    parser.add_argument("--producer-revision", required=True)
    args = parser.parse_args(argv)
    try:
        projection = consumer.read_projection(
            args.snapshot, args.expected_lock_sha256, args.expected_snapshot_sha256)
        proposal = build_proposal(
            projection,
            target_concept_id=args.concept_id,
            language=args.language,
            observation_type=args.observation_type,
            candidate_term=args.term,
            source_system=args.source_system,
            evidence_sha256=args.evidence_sha256,
            producer_revision=args.producer_revision,
            snapshot_sha256=args.expected_snapshot_sha256)
    except (ProposalRejected, consumer.Rejected) as exc:
        print(str(exc), file=sys.stderr)
        return 3
    if proposal is None:
        print(json.dumps({"status": "ALREADY_OBSERVED_NO_PROPOSAL",
                          "authority": "RESEARCH_PROPOSAL_ONLY"}, sort_keys=True))
        return 0
    print(json.dumps(proposal, sort_keys=True, ensure_ascii=False,
                     separators=(",", ":"), allow_nan=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
