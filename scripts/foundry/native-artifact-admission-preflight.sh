#!/usr/bin/env bash
# Public-only product/Foundry contract check. No image push, native target or live corpus.
set -Eeuo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
APPS="$ROOT/truenas-apps"
FOUNDRY="$ROOT/foundry-public"
UPSTREAM="77058ecb5b28199f35150912cacb13283a6fb244"
FOUNDRY_SHA="79545f02b9f88d1b0d6f6c0c12da1d063fd94388"
WORK="$(mktemp -d "$RUNNER_TEMP/bce-foundry-contract-XXXXXX")"
export WORK
cleanup() { local rc=$?; trap - EXIT; rm -rf "$WORK"; exit "$rc"; }
trap cleanup EXIT

test "$(git -C "$APPS" rev-parse HEAD)" = "$UPSTREAM" || { echo "Wrong pinned iX source"; exit 1; }
test "$(git -C "$FOUNDRY" rev-parse HEAD)" = "$FOUNDRY_SHA" || { echo "Wrong Foundry packager"; exit 1; }
test "$(git -C "$ROOT" rev-parse HEAD)" = "$GITHUB_SHA" || { echo "Product source checkout mismatch"; exit 1; }
cp -a "$ROOT/ix-dev/community/bitmagnet-concept-explorer" "$APPS/ix-dev/community/"
APP="$APPS/ix-dev/community/bitmagnet-concept-explorer"
python3 -m venv "$WORK/venv"
"$WORK/venv/bin/python" -m pip install --disable-pip-version-check --quiet 'PyYAML==6.0.2'
(
  cd "$APPS"
  "$WORK/venv/bin/python" .github/scripts/ci.py --app bitmagnet-concept-explorer \
    --train community --test-file loopback-published-values.yaml --render-only=true --skip-metadata
)
docker compose -p bce-source-contract -f "$APP/templates/rendered/docker-compose.yaml" config --format json > "$WORK/normalized.json"

# This must be the real unpublished candidate, not a synthetic positive fixture.
"$WORK/venv/bin/python" - <<'PY'
import json, os
from pathlib import Path
root = Path(os.environ["WORK"])
obj = json.loads((root / "normalized.json").read_text())
services = obj.get("services", {})
app = services["bitmagnet-concept-explorer"]
image = str(app["image"])
assert image.startswith("ghcr.io/mark-e-deyoung/bitmagnet-concept-explorer:")
assert "@sha256:" not in image, "Unexpected registry distribution pin; re-review gates"
assert image.endswith(":0.0.3-rc.1"), image
assert app.get("read_only") is True
assert str(app.get("user")) in ("65532", "65532:65532")
assert all(p.get("host_ip") == "127.0.0.1" for p in app.get("ports", []))
(root / "source-snapshot.json").write_text(json.dumps({
    "source_image": image, "runtime_shape": "source-only-not-deployable",
    "graph_authority": "unadmitted", "registry_digest": None
}, sort_keys=True))
print("BCE_SOURCE_RENDER_PASS: actual candidate is tag-only and cannot be promoted")
PY

# Fail-closed negative: a mutable candidate must not yield a native artifact.
set +e
python3 "$FOUNDRY/tools/package_truenas_deployment_artifact.py" \
  --app-name bce-rdte-probe --compose "$WORK/normalized.json" \
  --out "$WORK/blocked-artifact.json" --require-immutable-images \
  > "$WORK/negative.out" 2> "$WORK/negative.err"
status=$?
set -e
test "$status" -eq 2 || { echo "Expected tag-only source rejection; got $status"; exit 1; }
test ! -e "$WORK/blocked-artifact.json" || { echo "Rejected artifact leaked"; exit 1; }
grep -q "immutable image required" "$WORK/negative.err" || { echo "Wrong rejection cause"; exit 1; }
echo "FOUNDRY_IMMUTABILITY_NEGATIVE_PASS: no target artifact for tag-only BCE"

# Independently prove only the packaging SHAPE with conspicuously fabricated digests.
# Never store or publish this generated artifact; it is not signed or pullable.
"$WORK/venv/bin/python" - <<'PY'
import json, os
from pathlib import Path
work=Path(os.environ["WORK"])
obj=json.loads((work / "normalized.json").read_text())
for service in obj["services"].values():
    old = service.get("image")
    if not isinstance(old, str) or not old:
        raise SystemExit("service missing image: fail closed")
    repo = old.split("@")[0].rsplit(":", 1)[0]
    service["image"] = repo + "@sha256:" + ("b" * 64)
(work / "synthetic-digest-shape.json").write_text(json.dumps(obj, sort_keys=True))
PY
python3 "$FOUNDRY/tools/package_truenas_deployment_artifact.py" \
  --app-name bce-rdte-probe --compose "$WORK/synthetic-digest-shape.json" \
  --out "$WORK/synthetic-artifact.json" --require-immutable-images \
  > "$WORK/positive.out"
"$WORK/venv/bin/python" - <<'PY'
import hashlib, json, os, stat
from pathlib import Path
work=Path(os.environ["WORK"])
artifact=work / "synthetic-artifact.json"
a=json.loads(artifact.read_text())
assert a["schema"]=="truenas-foundry-deployment-artifact/v1"
assert a["runtime_target"]=="truenas-scale-apps"
assert a["app_name"]=="bce-rdte-probe"
assert a["create_payload"]["custom_app"] is True
assert a["materialization_identity"]=="sha256:"+a["compose_sha256"]
assert stat.S_IMODE(artifact.stat().st_mode)==0o600
assert all("@sha256:" in value["image"] for value in a["create_payload"]["custom_compose_config"]["services"].values())
print("FOUNDRY_ARTIFACT_SHAPE_ONLY_PASS: fabricated digest, no real pull/admission/apply")
PY
echo "BCE_FOUNDRY_CONSUMER_CONTRACT_PASS (source render + mandatory reject + synthetic-only artifact shape)"
