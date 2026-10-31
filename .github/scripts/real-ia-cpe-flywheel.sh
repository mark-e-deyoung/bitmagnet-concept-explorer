#!/usr/bin/env bash
# One exact Internet Archive item, actual Bitmagnet FTS + actual SearXNG
# Archive AdvancedSearch engine. Research-only CPE proposal, no authority
# promotion. The two observations share one original Archive ancestry.
set -Eeuo pipefail
: "$OUT"
: "$RDTE_DIR"
: "$AUTH"
F="$RDTE_DIR/internet-archive-flywheel"
mkdir -p "$F"
ENTRY="$(jq -c '[.entries[]|select(.source|startswith("internet-archive:"))][0] // empty' "$RDTE_DIR/manifest.json")"
if [[ -z "$ENTRY" ]]; then
  echo "::error::No individually license-claim-qualified Internet Archive metainfo admitted; specific Archive flywheel is NOT_RUN"
  exit 1
fi
ID="$(jq -r '.source_record' <<<"$ENTRY")"
HASH="$(jq -r '.infohash' <<<"$ENTRY")"
META_SHA="$(jq -r '.metainfo_sha256' <<<"$ENTRY")"
RIGHTS="$(jq -r '.rights_source_claim' <<<"$ENTRY")"
test -n "$ID" && test -n "$HASH"
echo "IA-LIVE-CPE: selected real Archive item=$ID infohash=$HASH metainfo_sha256=$META_SHA"

# The local native full-text query and external SearXNG query use the same
# original Archive identifier, not synthetic torrent IDs.
jq -n --arg q "$ID" '{
 query:$q, include_searxng:true, external_consent:true,
 max_bitmagnet_queries:1,max_searxng_queries:1
}' >"$F/request.json"
BASE=http://127.0.0.1:3086
curl --noproxy '*' -fsS --max-time 8 -u "$AUTH" -H 'Content-Type: application/json' \
  --data-binary @"$F/request.json" "$BASE/api/v1/sweeps/plan" -o "$F/plan.json"
jq -e '.read_only==true and .external_admitted==true and
  (.bitmagnet_queries|length)==1 and (.searxng_queries|length)==1' "$F/plan.json" >/dev/null

curl --noproxy '*' -fsS --max-time 80 -u "$AUTH" -H 'Content-Type: application/json' \
  --data-binary @"$F/request.json" "$BASE/api/v1/sweeps/execute" -o "$F/execution.json"
jq -e --arg hash "$HASH" --arg id "$ID" '
 .state=="complete" and .cpe_write==false and
 (.bitmagnet.read_only==true) and
 (any(.bitmagnet.results[]; .info_hash==$hash)) and
 (any(.external[].hits[]; .url==("https://archive.org/details/"+$id)))
' "$F/execution.json" >/dev/null
echo "IA-LIVE-CPE: exact genuine imported Archive infohash returned from real Bitmagnet AND original Archive identifier returned from real SearXNG official API search"

# Both observations derive from the SAME Archive object. Correct the
# ancestry in the proposed research packet to prevent false independent
# corroboration even if the two execution engines are different.
SOURCE_ANCESTRY="$(printf 'internet-archive:source-record:%s' "$ID" | sha256sum | cut -d' ' -f1)"
jq -n --slurpfile sweep "$F/execution.json" --arg ancestry "sha256:$SOURCE_ANCESTRY" '{
 proposal_kind:"source_mapping_candidate",
 candidate_label:"Archive BitTorrent",
 language:"en",
 concept_id:"public:source-record",
 evidence:($sweep[0].evidence_refs | map(.ancestry_ref=$ancestry) | .[:12])
}' >"$F/proposal-request.json"
curl --noproxy '*' -fsS --max-time 8 -u "$AUTH" -H 'Content-Type: application/json' \
 --data-binary @"$F/proposal-request.json" "$BASE/api/v1/sweeps/discovery/preview" -o "$F/proposal.json"
jq -e '
 .schema_version=="cpe-discovery-proposal-research-v1" and
 .authority=="not_accepted_not_transmitted" and
 .state=="local-review-export-only" and
 .distinct_ancestry_count==1 and (.evidence|length)>=2
' "$F/proposal.json" >/dev/null

jq -n --arg id "$ID" --arg hash "$HASH" --arg metainfo_sha256 "$META_SHA" \
  --arg rights "$RIGHTS" --arg head "$(git rev-parse HEAD)" \
  --slurpfile sweep "$F/execution.json" --slurpfile proposal "$F/proposal.json" '{
 schema_version:"public-ia-dual-track-cpe-research-v1",
 git_commit:$head, real_archive_record:$id, real_infohash:$hash,
 metainfo_sha256:$metainfo_sha256, rights_source_claim:$rights,
 rights_status:"source_claim_not_independently_adjudicated",
 evidence_ancestry_distinct:1,
 native_hits:($sweep[0].bitmagnet.results|length),
 external_hits:([$sweep[0].external[].hits[]]|length),
 proposal_id:$proposal[0].proposal_id,
 proposal_state:$proposal[0].state,
 cpe_promotion:false,
 independent_blind_reference:"NOT_RUN",
 conclusion:"Real dual-query same Archive source accepted as source-mapping research only; not independent semantic corroboration or rights clearance."
}' >"$F/metrics.json"
echo "IA-LIVE-CPE: real dual-track Archive source-mapping proposal PASS, unaccepted and single-source ancestry"
