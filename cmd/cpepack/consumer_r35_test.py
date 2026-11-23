"""R35 independent Python read-only consumer tests: original public synthetic only."""
from __future__ import annotations

import hashlib
import json
from pathlib import Path
import tempfile
import unittest

import consumer_r35 as r35

FIXTURE = Path(__file__).resolve().parents[2] / "testdata/cpepack/expected-public-projection.json"
EXPECTED_LOCK = "9b0d9d3bb07c81089ba65b7386931f8d0b71f66ccfd82042db3e530a50a49583"


def load_original():
    raw = FIXTURE.read_bytes()
    return r35.read_projection(FIXTURE, EXPECTED_LOCK, hashlib.sha256(raw).hexdigest())


def arbitrary_source(raw: bytes, lock: str = EXPECTED_LOCK, exact: str | None = None):
    with tempfile.TemporaryDirectory() as directory:
        path = Path(directory) / "untrusted.json"
        path.write_bytes(raw)
        return r35.read_projection(path, lock, exact or hashlib.sha256(raw).hexdigest())


class IndependentConsumerTests(unittest.TestCase):
    def test_original_fixture_id_and_multilingual_exact_evidence(self):
        p = load_original()
        self.assertEqual(set(p["concepts"]), {"public:work", "public:literary-work"})
        th = r35.query(p, "public:work", "th")
        self.assertEqual(th["status"], "OBSERVED")
        self.assertEqual(th["lexicalizations"], [{"language": "th", "term": "งาน"}])
        de = r35.query(p, "public:literary-work", "de")
        self.assertEqual(de["lexicalizations"], [{"language": "de", "term": "Werk"}])
        self.assertEqual(de["authority"], "research_only")

    def test_unknown_concept_and_missing_language_do_not_infer_equivalence(self):
        p = load_original()
        self.assertEqual(r35.query(p, "public:unknown", None)["status"], "UNKNOWN_CONCEPT")
        th = r35.query(p, "public:literary-work", "th")
        self.assertEqual(th["status"], "NO_LEXICALIZATION")
        self.assertEqual(th["lexicalizations"], [])
        self.assertNotEqual(th["status"], "NO_EQUIVALENT")

    def test_wrong_independent_lock_fails(self):
        raw = FIXTURE.read_bytes()
        with self.assertRaisesRegex(r35.Rejected, "SOURCE_LOCK_MISMATCH"):
            arbitrary_source(raw, lock="0"*64)

    def test_mutated_projection_bytes_rejected_even_with_same_lock_text(self):
        raw = FIXTURE.read_bytes()
        altered = raw.replace(b"A creative work.", b"An unapproved replacement.", 1)
        self.assertNotEqual(raw, altered)
        with self.assertRaisesRegex(r35.Rejected, "SNAPSHOT_BYTES_MISMATCH"):
            arbitrary_source(altered, exact=hashlib.sha256(raw).hexdigest())

    def test_self_declared_upgrade_rejected_even_when_new_digest_supplied(self):
        raw = FIXTURE.read_bytes().replace(b'"research_only"', b'"accepted"', 1)
        with self.assertRaisesRegex(r35.Rejected, "UNADMITTED_SNAPSHOT_AUTHORITY"):
            arbitrary_source(raw)

    def test_duplicate_concept_id_rejected(self):
        d = json.loads(FIXTURE.read_text("utf-8"))
        d["concepts"].append(d["concepts"][0])
        raw = json.dumps(d, ensure_ascii=False).encode("utf-8")
        with self.assertRaisesRegex(r35.Rejected, "DUPLICATE_CONCEPT_ID"):
            arbitrary_source(raw)

    def test_duplicate_json_member_rejected(self):
        raw = b'{"schema_version":"cpe-readmodel-v0","schema_version":"cpe-readmodel-v0","authority":"research_only","lock_sha256":"' + EXPECTED_LOCK.encode() + b'","concepts":[]}'
        with self.assertRaisesRegex(r35.Rejected, "DUPLICATE_JSON_KEY"):
            arbitrary_source(raw)

    def test_malformed_utf8_rejected(self):
        raw = FIXTURE.read_bytes() + b'\xff'
        with self.assertRaisesRegex(r35.Rejected, "MALFORMED_SNAPSHOT"):
            arbitrary_source(raw)

    def test_unpaired_json_surrogate_rejected_as_string(self):
        d = json.loads(FIXTURE.read_text("utf-8"))
        d["concepts"][0]["definition"] = "\ud800"
        raw = json.dumps(d, ensure_ascii=True).encode("utf-8")
        with self.assertRaisesRegex(r35.Rejected, "INVALID_DEFINITION"):
            arbitrary_source(raw)

    def test_relation_to_nonexistent_concept_refused(self):
        d = json.loads(FIXTURE.read_text("utf-8"))
        d["concepts"][0]["relations"][0]["target_id"] = "public:missing"
        raw = json.dumps(d).encode("utf-8")
        with self.assertRaisesRegex(r35.Rejected, "UNRESOLVED_CONCEPT_RELATION"):
            arbitrary_source(raw)

    def test_oversized_byte_input_rejected(self):
        raw = b' '*(r35.MAX_BYTES+1)
        with self.assertRaisesRegex(r35.Rejected, "OVERSIZED_SNAPSHOT"):
            arbitrary_source(raw)


if __name__ == "__main__":
    unittest.main()
