#!/usr/bin/env python3
"""Extract a small, provenance-recorded real Project Gutenberg catalog sample.

The supplied input is the *official bulk catalog* (never per-ebook scraping).
Output is ONLY metadata. Torrent infohashes are deliberately artificial
per-record test identifiers; they do not identify real torrents or content.
No book text, artwork, torrents, or copyright/rights conclusions are imported.
"""
import argparse
import csv
import gzip
import hashlib
import json
import pathlib
import re
import sys

ANCHORS = ("62", "55", "11", "35")
BUDGET = {"en": 144, "de": 24, "fr": 24, "es": 24, "other": 16}
SEED = "bce-real-public-catalog-rep-20261031-v1"
SOURCE = "Project Gutenberg bulk CSV metadata; research only"


def provenance_hash(value):
    return hashlib.sha256(value).hexdigest()


def field(record, *names):
    normalized = {
        re.sub(r"[^a-z0-9]", "", k.lower()): (v or "").strip()
        for k, v in record.items()
        if isinstance(k, str)
    }
    for name in names:
        value = normalized.get(re.sub(r"[^a-z0-9]", "", name.lower()), "")
        if value:
            return " ".join(value.split())
    return ""


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--catalog", type=pathlib.Path, required=True)
    parser.add_argument("--output", type=pathlib.Path, required=True)
    parser.add_argument("--manifest", type=pathlib.Path, required=True)
    args = parser.parse_args()

    raw = args.catalog.read_bytes()
    if len(raw) < 100_000 or len(raw) > 15_000_000:
        raise ValueError("official bulk catalog expected, compressed size outside 100KB–15MB")
    catalog_sha = provenance_hash(raw)
    entries = {}
    scanned = 0
    with gzip.open(args.catalog, mode="rt", encoding="utf-8-sig", newline="") as catalog:
        reader = csv.DictReader(catalog)
        if reader.fieldnames is None:
            raise ValueError("bulk catalog missing header")
        for record in reader:
            scanned += 1
            bookid = field(record, "Text#", "Text", "Ebook No.", "Ebook ID")
            title = field(record, "Title")
            kind = field(record, "Type")
            language = field(record, "Language")
            author = field(record, "Authors", "Author")
            if not bookid.isascii() or not bookid.isdigit() or kind.lower() != "text":
                continue
            if not title or not language or len(title) > 400 or len(author) > 200:
                continue
            if not re.match(r"^[a-z]{2}(?:$|[;, ])", language):
                continue
            lang = re.split(r"[;,\s]+", language.lower(), maxsplit=1)[0]
            bucket = lang if lang in BUDGET and lang != "other" else "other"
            if bookid in entries:
                raise ValueError("duplicate Gutenberg eBook id in bulk catalog: " + bookid)
            entries[bookid] = {
                "gutenberg_ebook_id": int(bookid),
                "title": title,
                "authors": author,
                "language": lang,
                "bucket": bucket,
                "gutenberg_ebook_release_date": field(record, "Issued"),
                "source_url": "https://www.gutenberg.org/ebooks/" + bookid,
                "metadata_status": "public_source_report_not_cpe_accepted",
            }

    if scanned < 10_000 or len(entries) < 10_000:
        raise ValueError("not a complete-looking official catalog")
    if not all(key in entries for key in ANCHORS):
        raise ValueError("one or more public Gutenberg lore pack anchor ids missing from catalog")

    selected = [entries[k] for k in ANCHORS]
    anchor_set = set(ANCHORS)
    counts = {bucket: 0 for bucket in BUDGET}
    for item in selected:
        counts[item["bucket"]] += 1
    for bucket, limit in BUDGET.items():
        choices = [
            (provenance_hash((SEED + ":" + str(item["gutenberg_ebook_id"])).encode()), item)
            for key, item in entries.items()
            if key not in anchor_set and item["bucket"] == bucket
        ]
        choices.sort(key=lambda pair: pair[0])
        selected.extend(item for _, item in choices[: max(0, limit - counts[bucket])])
    selected.sort(key=lambda item: item["gutenberg_ebook_id"])

    if len(selected) < 180:
        raise ValueError("catalog language/sample strata too small")
    import_records = []
    for item in selected:
        # This is intentionally not a real torrent hash; do not use for peers.
        artificial_hash = hashlib.sha1(
            ("BCE-NOT-A-REAL-TORRENT:" + str(item["gutenberg_ebook_id"])).encode()
        ).hexdigest()
        item["artificial_infohash_for_disposable_test_only"] = artificial_hash
        title = item["title"]
        author = item["authors"]
        import_records.append({
            "source": "bce-public-gutenberg-catalog-only",
            "infoHash": artificial_hash,
            "name": ("CATALOG-ONLY / NOT A TORRENT | " + title + " | " + author)[:480],
            "size": len(title.encode("utf-8")) + 4096,  # fabricated test fixture value
            "contentType": "ebook",
            "title": title,
        })
    args.output.write_text(
        "".join(json.dumps(r, ensure_ascii=False, separators=(",", ":")) + "\n" for r in import_records),
        encoding="utf-8",
    )
    manifest = {
        "schema_version": "bce-real-upstream-catalog-metadata-rep-v1",
        "status": "research_only_not_cpe_accepted_or_real_torrent",
        "provenance": SOURCE,
        "source_url": "https://www.gutenberg.org/cache/epub/feeds/pg_catalog.csv.gz",
        "catalog_gzip_sha256": catalog_sha,
        "sample_method": SEED,
        "records_in_catalog": scanned,
        "selected": len(selected),
        "language_strata": {k: sum(i["bucket"] == k for i in selected) for k in BUDGET},
        "anchor_ids": list(map(int, ANCHORS)),
        "no_text_or_artwork": True,
        "fabricated_fields": ["infoHash", "size", "name prefix"],
        "items": selected,
    }
    args.manifest.write_text(
        json.dumps(manifest, ensure_ascii=False, sort_keys=True, indent=2) + "\n",
        encoding="utf-8",
    )
    print(
        "PUBLIC-GUTENBERG-CATALOG: "
        + f"metadata_sha256={catalog_sha} scanned={scanned} selected={len(selected)} "
        + f"strata={manifest['language_strata']}"
    )


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print("PUBLIC-GUTENBERG-CATALOG: FAIL " + str(exc), file=sys.stderr)
        sys.exit(1)
