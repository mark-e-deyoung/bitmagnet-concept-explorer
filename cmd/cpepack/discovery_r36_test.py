"""R36 synthetic research ConceptDiscoveryProposal tests; no semantic acceptance."""
from __future__ import annotations

import hashlib
import json
from pathlib import Path
import tempfile
import unittest

import consumer_r35 as consumer
import discovery_r36 as discovery

FIXTURE = Path(__file__).resolve().parents[2] / "testdata/cpepack/expected-public-projection.json"
SOURCE_LOCK = "9b0d9d3bb07c81089ba65b7386931f8d0b71f66ccfd82042db3e530a50a49583"
PRODUCER_REVISION = "3e66404fd0188afd4c809441827f9fe3ea585cb2"
EVIDENCE = hashlib.sha256(b"original-synthetic-source-evidence").hexdigest()


def fixture():
    raw = FIXTURE.read_bytes()
    source = consumer.read_projection(FIXTURE, SOURCE_LOCK, hashlib.sha256(raw).hexdigest())
    return source, hashlib.sha256(raw).hexdigest()


def make(**overrides):
    source, exact_bytes = fixture()
    params = {
        "target_concept_id": "public:literary-work",
        "language": "th",
        "observation_type": "native_source_term",
        "candidate_term": "งานวรรณกรรม",
        "source_system": "fixture.public.lexical.source",
        "evidence_sha256": EVIDENCE,
        "producer_revision": PRODUCER_REVISION,
        "snapshot_sha256": exact_bytes,
    }
    params.update(overrides)
    return discovery.build_proposal(source, **params)


class DiscoveryR36Tests(unittest.TestCase):
    def test_missing_lexicalization_yields_unadjudicated_candidate_only(self):
        p = make()
        self.assertEqual(p["schema_version"], discovery.VERSION)
        self.assertEqual(p["proposal_kind"], "lexicalization_candidate")
        self.assertEqual(p["authority"], "RESEARCH_PROPOSAL_ONLY")
        self.assertEqual(p["adjudication"], "UNADJUDICATED")
        self.assertEqual(p["mapping"]["equivalence_status"], "UNKNOWN")
        self.assertTrue(p["mapping"]["candidate_not_accepted"])
        self.assertEqual(p["observation"]["native_term"], "งานวรรณกรรม")
        self.assertEqual(p["source_projection"]["source_lock_sha256"], SOURCE_LOCK)
        self.assertEqual(p["evidence"]["source_ancestry"], [EVIDENCE])
        self.assertEqual(p["evidence"]["support_count"], 1)
        self.assertEqual(p["evidence"]["independent_corroboration"], "NOT_ESTABLISHED")
        self.assertFalse(p["privacy"]["sanitization_independently_qualified"])

    def test_identical_observation_is_deterministic_and_not_false_corroboration(self):
        a = make()
        b = make()
        self.assertEqual(a, b)
        self.assertTrue(a["proposal_id"].startswith("proposal:sha256:"))
        self.assertEqual(a["evidence"]["support_count"], 1)
        self.assertEqual(len(a["evidence"]["source_ancestry"]), 1)

    def test_already_known_exact_native_term_produces_no_proposal(self):
        self.assertIsNone(make(language="de", candidate_term="Werk"))

    def test_consumer_miss_on_known_term_is_capability_gap_not_new_concept(self):
        p = make(language="de", observation_type="consumer_lookup_miss", candidate_term=None)
        self.assertEqual(p["proposal_kind"], "coverage_gap")
        self.assertEqual(p["mapping"]["classification"], "KNOWN_CONCEPT_RETRIEVAL_MISS_NOT_ONTOLOGY_GAP")
        self.assertEqual(p["mapping"]["equivalence_status"], "UNKNOWN")

    def test_unknown_concept_is_not_automatically_novel(self):
        p = make(target_concept_id="public:unregistered")
        self.assertEqual(p["proposal_kind"], "coverage_gap")
        self.assertEqual(p["mapping"]["classification"], "UNKNOWN_CONCEPT_NOT_PROOF_OF_NOVELTY")
        self.assertEqual(p["observation"]["consumer_status"], "UNKNOWN_CONCEPT")
        self.assertNotIn("new_concept_candidate", json.dumps(p))

    def test_no_lexicalization_does_not_claim_hard_negative(self):
        p = make()
        self.assertEqual(p["observation"]["consumer_status"], "NO_LEXICALIZATION")
        self.assertNotEqual(p["mapping"]["equivalence_status"], "NO_EQUIVALENT")

    def test_source_revision_and_evidence_required_exact(self):
        for key, value, expected in [
            ("producer_revision", "short", "INVALID_PRODUCER_REVISION"),
            ("evidence_sha256", "0"*63, "INVALID_EVIDENCE_DIGEST"),
            ("snapshot_sha256", "m"*64, "INVALID_SNAPSHOT_DIGEST"),
        ]:
            with self.subTest(key=key):
                with self.assertRaisesRegex(discovery.ProposalRejected, expected):
                    make(**{key: value})

    def test_private_or_unreviewed_privacy_class_refused(self):
        with self.assertRaisesRegex(discovery.ProposalRejected, "UNQUALIFIED_PRIVACY_CLASS"):
            make(privacy_class="private_unsanitized")

    def test_raw_urls_or_paths_and_malformed_unicode_refused(self):
        for value, error in [
            ("https://example.invalid/raw", "POTENTIAL_RAW_LOCATOR"),
            ("some/local/file", "POTENTIAL_RAW_LOCATOR"),
            ("a"*97, "INVALID_NATIVE_TERM"),
            ("\ud800", "INVALID_NATIVE_TERM"),
            ("hello\nraw", "INVALID_NATIVE_TERM"),
        ]:
            with self.subTest(candidate=value):
                with self.assertRaisesRegex(discovery.ProposalRejected, error):
                    make(candidate_term=value)

    def test_invalid_observation_type_locale_and_missing_term_refused(self):
        cases = [
            ({"observation_type": "accept"}, "INVALID_OBSERVATION_TYPE"),
            ({"language": "th/../private"}, "INVALID_LANGUAGE"),
            ({"observation_type": "consumer_lookup_miss"}, "UNEXPECTED_NATIVE_TERM"),
            ({"candidate_term": None}, "INVALID_NATIVE_TERM"),
            ({"source_system": "https://private.example"}, "INVALID_SOURCE_SYSTEM"),
        ]
        for kwargs, expected in cases:
            with self.subTest(kwargs=repr(kwargs)):
                with self.assertRaisesRegex(discovery.ProposalRejected, expected):
                    make(**kwargs)

    def test_tampered_bytes_fail_before_proposal_creation(self):
        original = FIXTURE.read_bytes()
        altered = original.replace(b"A creative work.", b"Unreviewed definition.", 1)
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp) / "tampered.json"
            path.write_bytes(altered)
            with self.assertRaisesRegex(consumer.Rejected, "SNAPSHOT_BYTES_MISMATCH"):
                consumer.read_projection(path, SOURCE_LOCK, hashlib.sha256(original).hexdigest())

    def test_proposal_is_plain_deterministic_json_not_accepted_artifact(self):
        p = make()
        b1 = json.dumps(p, sort_keys=True, ensure_ascii=False, separators=(",", ":")).encode("utf-8")
        b2 = json.dumps(make(), sort_keys=True, ensure_ascii=False, separators=(",", ":")).encode("utf-8")
        self.assertEqual(b1, b2)
        self.assertNotIn(b'"accepted"', b1)
        self.assertIn("งานวรรณกรรม".encode("utf-8"), b1)


if __name__ == "__main__":
    unittest.main()
