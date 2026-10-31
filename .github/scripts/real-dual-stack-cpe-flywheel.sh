#!/usr/bin/env bash
# REAL native Bitmagnet + real SearXNG + public CPE research-only flywheel.
set -Eeuo pipefail
: "$OUT"
: "$RDTE_DIR"
: "$AUTH"
BASE='http://127.0.0.1:3086'
F="$RDTE_DIR/flywheel"
mkdir -p "$F"
GRAPH_HASH="$(sha256sum testdata/public-cpe-packs/gutenberg-lore-v0/graph-preview.json | cut -d' ' -f1)"

ready=0
for i in $(seq 1 30); do
  if curl --noproxy '*' -fsS --max-time 10 \
    --get --data-urlencode 'q=A Princess of Mars' \
    --data-urlencode 'format=json' \
    'http://127.0.0.1:8888/search' -o "$F/searxng-preflight.json" 2>/dev/null \
    && jq -e '.results | type == "array"' "$F/searxng-preflight.json" >/dev/null 2>&1; then
    ready=1; break
  fi
  sleep 2
done
if [[ "$ready" != "1" ]]; then
  echo "::error::Real SearXNG did not serve valid JSON"
  exit 1
fi
echo "CPE-LIVE-SWEEP: real SearXNG JSON results=$(jq -r '.results|length' "$F/searxng-preflight.json")"

cat >"$F/request.json" <<'QUERY'
{"query":"Barsoom","concept_ids":["public:story-world"],"languages":["en","de","th"],"include_searxng":true,"external_consent":true,"max_bitmagnet_queries":5,"max_searxng_queries":2}
QUERY
curl --noproxy '*' -fsS --max-time 10 -u "$AUTH" -H 'Content-Type: application/json' \
  --data-binary @"$F/request.json" "$BASE/api/v1/sweeps/plan" -o "$F/plan.json"
jq -e --arg sha "sha256:$GRAPH_HASH" '
 .read_only == true and .external_admitted == true and .projection_digest == $sha and
 (.bitmagnet_queries|length)>=2 and (.searxng_queries|length)==2
' "$F/plan.json" >/dev/null
echo "CPE-LIVE-SWEEP: real planner admitted native and independent external source tracks"

curl --noproxy '*' -fsS --max-time 80 -u "$AUTH" -H 'Content-Type: application/json' \
  --data-binary @"$F/request.json" "$BASE/api/v1/sweeps/execute" -o "$F/execution.json"
jq -e '
 (.state == "complete" or .state == "partial") and .cpe_write == false and
 .plan.external_admitted == true and .plan.read_only == true and
 .bitmagnet.read_only == true and (.bitmagnet.results|length)>=1 and
 (.external|length)>=1 and ([.external[].hits[]]|length)>=1 and
 (.evidence_refs|length)>=2
' "$F/execution.json" >/dev/null
echo "CPE-LIVE-SWEEP: Bitmagnet hits=$(jq '.bitmagnet.results|length' "$F/execution.json") SearXNG hits=$(jq '[.external[].hits[]]|length' "$F/execution.json")"
jq -c '{state,failures,source_counts:{bitmagnet:(.bitmagnet.results|length),external:([.external[].hits[]]|length)},research_proposals:(.research_proposals|length),authority:.plan.authority}' "$F/execution.json"

# Existing CPE #202 research-only API. This source-mapping candidate is
# nominated by the analyst, NOT corroborated or accepted by the engine.
jq -n --slurpfile sweep "$F/execution.json" --arg digest "sha256:$GRAPH_HASH" '{
 proposal_kind:"source_mapping_candidate",
 candidate_label:"Archive BitTorrent",
 language:"en",
 concept_id:"public:source-record",
 projection_digest:$digest,
 evidence:($sweep[0].evidence_refs|.[:8])
}' >"$F/proposal-request.json"
curl --noproxy '*' -fsS --max-time 8 -u "$AUTH" \
 -H 'Content-Type: application/json' --data-binary @"$F/proposal-request.json" \
 "$BASE/api/v1/sweeps/discovery/preview" -o "$F/proposal.json"
jq -e '
 .schema_version=="cpe-discovery-proposal-research-v1" and
 .authority=="not_accepted_not_transmitted" and
 .state=="local-review-export-only" and
 .proposal_kind=="source_mapping_candidate" and (.evidence|length)>=1
' "$F/proposal.json" >/dev/null

cat >"$F/native-baseline.json" <<'JSON'
{"query":"Barsoom"}
JSON
cat >"$F/research-expansion.json" <<'JSON'
{"query":"Barsoom","expansions":[{"term":"A Princess of Mars","provenance":"unverified-public-source:gutenberg:62"}]}
JSON
for name in native-baseline research-expansion; do
  curl --noproxy '*' -fsS --max-time 25 -u "$AUTH" \
    -H 'Content-Type: application/json' --data-binary @"$F/$name.json" \
    "$BASE/api/v1/query/execute" -o "$F/$name-result.json"
  jq -e '.state=="complete" and .read_only==true' "$F/$name-result.json" >/dev/null
done
ANCHOR_HASH="$(jq -r '.items[] | select(.gutenberg_ebook_id == 62) | .artificial_infohash_for_disposable_test_only' "$OUT/public-catalog-manifest.json")"
jq -e --arg hash "$ANCHOR_HASH" \
 '[.results[]|select(.info_hash==$hash)]|length==0' "$F/native-baseline-result.json" >/dev/null
jq -e --arg hash "$ANCHOR_HASH" \
 '[.results[]|select(.info_hash==$hash and (.tracks|index("concept")!=null))]|length>=1' "$F/research-expansion-result.json" >/dev/null
jq -n --arg head "$(git rev-parse HEAD)" --arg graph_sha "$GRAPH_HASH" \
 --slurpfile baseline "$F/native-baseline-result.json" \
 --slurpfile expanded "$F/research-expansion-result.json" \
 --slurpfile sweep "$F/execution.json" \
 --slurpfile proposal "$F/proposal.json" \
 --slurpfile ia "$RDTE_DIR/manifest.json" '{
  schema_version:"cpe-rdte-dual-stack-flywheel-research-v1",
  git_commit:$head,
  graph_sha256:$graph_sha,
  authority:"no_cpe_promotion",
  origin:"actual Bitmagnet native FTS and independent real SearXNG",
  independent_blind_reference:"NOT_RUN",
  internet_archive_admitted:([$ia[0].entries[]|select(.source|startswith("internet-archive:"))]|length),
  baseline_results:($baseline[0].results|length),
  candidate_expansion_results:($expanded[0].results|length),
  external_results:([$sweep[0].external[].hits[]]|length),
  research_source_candidate_match_gutenberg62:true,
  proposal_id:$proposal[0].proposal_id,
  research_proposal_state:$proposal[0].state,
  interpretation:"Search hypothesis adds a Gutenberg record; not accepted concept or independent recall gain. Held-out evaluation and CPE #202 adjudication remain necessary."
}' >"$F/metrics.json"
echo "CPE-LIVE-SWEEP: native and external real sweep with nonnormative proposal PASS; IA admissible torrents=$(jq -r '.internet_archive_admitted' "$F/metrics.json")"
