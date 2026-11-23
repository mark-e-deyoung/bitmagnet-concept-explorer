#!/usr/bin/env bash
# Public synthetic only. Upstream TrueNAS source render -> graph mount -> live Explorer API.
# Does not prove an admitted CPE snapshot, native Foundry apply, or physical TrueNAS.
set -Eeuo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
APPS="$ROOT/truenas-apps"
APP="$APPS/ix-dev/community/bitmagnet-concept-explorer"
PIN="77058ecb5b28199f35150912cacb13283a6fb244"
IMAGE="ghcr.io/mark-e-deyoung/bitmagnet-concept-explorer:0.0.1"
WORK="$(mktemp -d "$RUNNER_TEMP/bce-graph-mount-XXXXXX")"
PROJECT="bce-graph-mount-$"
# TrueNAS ix-lib host-path validator admits /mnt/... only, not $RUNNER_TEMP.
GRAPH_DIR="/mnt/bce-graph-fixture-$"
GRAPH_FILE="$GRAPH_DIR/graph.json"
export GRAPH_FILE
COMPOSE="$WORK/rendered.yaml"
BASE_URL="http://127.0.0.1:3086"
export WORK APP
cleanup() {
  code=$?
  trap - EXIT
  if [[ -f "$COMPOSE" ]]; then
    docker compose -p "$PROJECT" -f "$COMPOSE" down -v --remove-orphans >/dev/null 2>&1 || true
  fi
  sudo rm -rf "$GRAPH_DIR"
  rm -rf "$WORK"
  exit "$code"
}
trap cleanup EXIT
test "$(git -C "$APPS" rev-parse HEAD)" = "$PIN" || { echo "Pinned upstream mismatch" >&2; exit 1; }
cp -a "$ROOT/ix-dev/community/bitmagnet-concept-explorer" "$APPS/ix-dev/community/"
sudo mkdir -p "$GRAPH_DIR"
sudo cp "$ROOT/fixtures/concept-graph.synthetic.json" "$GRAPH_FILE"
sudo chmod 755 "$GRAPH_DIR"
sudo chmod 644 "$GRAPH_FILE"
export GRAPH_HASH="$(sha256sum "$GRAPH_FILE" | cut -d' ' -f1)"
python3 -m venv "$WORK/venv"
"$WORK/venv/bin/python" -m pip install --disable-pip-version-check --quiet pyyaml
"$WORK/venv/bin/python" - <<'PY'
import os
from pathlib import Path
base = Path(os.environ["APP"]) / "templates/test_values/loopback-published-values.yaml"
out = base.with_name("runtime-graph-values.yaml")
src = base.read_text(encoding="utf-8")
if "research_data:" in src:
    raise SystemExit("Unexpected research_data in immutable loopback fixture")
out.write_text(src + "\nresearch_data:\n"
  + "  graph_file: " + os.environ["GRAPH_FILE"] + "\n"
  + "  graph_sha256: " + os.environ["GRAPH_HASH"] + "\n"
  + "  evidence_file: ''\n  evidence_sha256: ''\n", encoding="utf-8")
PY
(
 cd "$APPS"
 "$WORK/venv/bin/python" .github/scripts/ci.py --app bitmagnet-concept-explorer \
   --train community --test-file runtime-graph-values.yaml --render-only=True --skip-metadata
)
cp "$APP/templates/rendered/docker-compose.yaml" "$COMPOSE"
"$WORK/venv/bin/python" - <<'PY'
import os
from pathlib import Path
import yaml
compose = yaml.safe_load(Path(os.environ["WORK"], "rendered.yaml").read_text())
svc = compose["services"]["bitmagnet-concept-explorer"]
env = svc["environment"]
if isinstance(env, list):
    env = dict(item.split("=", 1) for item in env)
assert env["BCE_CPE_GRAPH_FILE"] == "/opt/bce/research/graph-preview.json"
assert env["BCE_CPE_GRAPH_SHA256"] == os.environ["GRAPH_HASH"]
assert svc.get("read_only") is True and str(svc["user"]) in ("65532", "65532:65532")
mounts = [v for v in svc.get("volumes", []) if v.get("target") == env["BCE_CPE_GRAPH_FILE"]]
assert len(mounts) == 1 and mounts[0]["source"] == os.environ["GRAPH_FILE"]
assert mounts[0]["type"] == "bind" and mounts[0]["read_only"] is True
ports = svc.get("ports", [])
assert ports and all((p.get("host_ip") if isinstance(p, dict) else str(p).split(":")[0]) == "127.0.0.1" for p in ports)
print("RENDERED_GRAPH_MOUNT_PASS")
PY
# A missing pinned builder must fail; do not switch to a floating image.
docker build -f "$ROOT/Containerfile" -t "$IMAGE" "$ROOT"
docker compose -p "$PROJECT" -f "$COMPOSE" config >/dev/null
docker compose -p "$PROJECT" -f "$COMPOSE" up -d --pull never
CID="$(docker compose -p "$PROJECT" -f "$COMPOSE" ps -q bitmagnet-concept-explorer)"
test -n "$CID"
docker inspect "$CID" | "$WORK/venv/bin/python" -c '
import json, sys
c = json.load(sys.stdin)[0]
m = [v for v in c["Mounts"] if v["Destination"] == "/opt/bce/research/graph-preview.json"]
assert len(m) == 1 and m[0]["RW"] is False
assert c["HostConfig"]["ReadonlyRootfs"] is True
print("RUNTIME_READONLY_GRAPH_MOUNT_PASS")
'
ready=0
for attempt in $(seq 1 30); do
  if curl --noproxy '*' -fsS --max-time 2 "$BASE_URL/healthz" >/dev/null 2>&1; then ready=1; break; fi
  sleep 1
done
test "$ready" = 1 || { docker compose -p "$PROJECT" -f "$COMPOSE" logs --tail=80; exit 1; }
status="$(curl --noproxy '*' -s -o /dev/null -w '%{http_code}' "$BASE_URL/api/v1/concepts")"
test "$status" = 401 || { echo "Expected 401 for unauthenticated concept API, got $status" >&2; exit 1; }
curl --noproxy '*' -fsS --max-time 10 -u 'fixture-user:fixture-password-not-for-deployment' \
  "$BASE_URL/api/v1/concepts" > "$WORK/concepts.json"
"$WORK/venv/bin/python" - <<'PY'
import json, os
from pathlib import Path
response = json.loads(Path(os.environ["WORK"], "concepts.json").read_text())
assert response.get("authority_status") == "pinned-preview-not-normative"
assert any(c.get("id") == "fixture.media" for c in response.get("concepts", []))
print("GRAPH_API_RUNTIME_PASS: synthetic graph loaded; authority provisional")
PY
docker compose -p "$PROJECT" -f "$COMPOSE" down -v --remove-orphans
test -z "$(docker ps -aq --filter name="$PROJECT")"
test -z "$(docker network ls -q --filter name="$PROJECT")"
echo "SYNTHETIC_GRAPH_MOUNT_F0_F3_PASS; native Foundry F4+, accepted CPE and hardware HELD"
