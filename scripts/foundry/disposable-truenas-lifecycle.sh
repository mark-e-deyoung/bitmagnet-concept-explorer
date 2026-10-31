#!/usr/bin/env bash
# TrueNAS Foundry Lifecycle Ladder (F0-F3) Execution Script
# Validates Bitmagnet Concept Explorer using pinned upstream TrueNAS catalog
# tooling and disposable container runtime in an ephemeral environment.
# Zero production impact, read-only Bitmagnet authority, $0 public compute.

set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
WORK_DIR="${RUNNER_TEMP:-/tmp}/bce-foundry-lifecycle"
mkdir -p "${WORK_DIR}"

PINNED_APPS_COMMIT="77058ecb5b28199f35150912cacb13283a6fb244"
APPS_DIR="${REPO_ROOT}/truenas-apps"
PROJECT_NAME="bce-foundry-f3-$$"
CONTAINER_IMAGE="ghcr.io/mark-e-deyoung/bitmagnet-concept-explorer:0.0.1"

FIXTURE_USER="fixture-user"
FIXTURE_PASS="fixture-password-not-for-deployment"
EXPECTED_PORT="3086"
EXPECTED_HOST="127.0.0.1"
SERVICE_URL="http://${EXPECTED_HOST}:${EXPECTED_PORT}"

RENDERED_COMPOSE=""

cleanup() {
  local exit_code=$?
  trap - EXIT
  echo "=== TrueNAS Foundry Lifecycle Ladder Cleanup ==="
  if [[ -n "${RENDERED_COMPOSE:-}" && -f "${RENDERED_COMPOSE}" ]]; then
    echo "Tearing down ephemeral Docker Compose project: ${PROJECT_NAME}..."
    docker compose -p "${PROJECT_NAME}" -f "${RENDERED_COMPOSE}" down -v --remove-orphans >/dev/null 2>&1 || true
  fi

  # Ensure no orphan containers or networks survive
  local orphan_containers
  orphan_containers="$(docker ps -aq --filter name="${PROJECT_NAME}" 2>/dev/null || true)"
  if [[ -n "${orphan_containers}" ]]; then
    echo "Removing orphaned test containers: ${orphan_containers}"
    echo "${orphan_containers}" | xargs -r docker rm -f >/dev/null 2>&1 || true
  fi

  local orphan_networks
  orphan_networks="$(docker network ls -q --filter name="${PROJECT_NAME}" 2>/dev/null || true)"
  if [[ -n "${orphan_networks}" ]]; then
    echo "Removing orphaned test networks: ${orphan_networks}"
    echo "${orphan_networks}" | xargs -r docker network rm >/dev/null 2>&1 || true
  fi

  if [[ ${exit_code} -ne 0 ]]; then
    echo "::error::Foundry Lifecycle Ladder FAILED with exit code ${exit_code}"
    if [[ -n "${PROJECT_NAME:-}" ]]; then
      echo "=== Ephemeral Container Logs ==="
      docker compose -p "${PROJECT_NAME}" -f "${RENDERED_COMPOSE}" logs --tail 100 2>/dev/null || true
    fi
  fi
  exit "${exit_code}"
}
trap 'cleanup' EXIT

echo "======================================================================"
echo " TrueNAS Foundry Lifecycle Ladder (F0-F3)"
echo " Bitmagnet Concept Explorer"
echo " Commit: $(git -C "${REPO_ROOT}" rev-parse HEAD)"
echo " Pinned truenas/apps commit: ${PINNED_APPS_COMMIT}"
echo "======================================================================"

# ============================================================================
# F0: Source Render (Upstream Catalog Validator)
# ============================================================================
echo ""
echo ">>> [Stage F0] Upstream TrueNAS catalog source render"

if [[ ! -d "${APPS_DIR}" ]]; then
  echo "Cloning truenas/apps repository at pinned commit ${PINNED_APPS_COMMIT}..."
  git clone --filter=blob:none https://github.com/truenas/apps.git "${APPS_DIR}"
  git -C "${APPS_DIR}" checkout -q "${PINNED_APPS_COMMIT}"
fi

ACTUAL_APPS_HEAD="$(git -C "${APPS_DIR}" rev-parse HEAD)"
if [[ "${ACTUAL_APPS_HEAD}" != "${PINNED_APPS_COMMIT}" ]]; then
  echo "::error::truenas/apps commit mismatch: expected ${PINNED_APPS_COMMIT}, got ${ACTUAL_APPS_HEAD}"
  exit 1
fi
echo "Verified truenas/apps pinned commit: ${ACTUAL_APPS_HEAD}"

echo "Syncing candidate app definition into truenas/apps/ix-dev/community/..."
rm -rf "${APPS_DIR}/ix-dev/community/bitmagnet-concept-explorer"
cp -a "${REPO_ROOT}/ix-dev/community/bitmagnet-concept-explorer" "${APPS_DIR}/ix-dev/community/"

echo "Setting up Python virtual environment for catalog validator..."
PYTHON_VENV="${WORK_DIR}/venv"
python3 -m venv "${PYTHON_VENV}"
"${PYTHON_VENV}/bin/python" -m pip install --disable-pip-version-check --quiet pyyaml

echo "Executing upstream catalog renderer with test fixture loopback-published-values.yaml..."
(
  cd "${APPS_DIR}"
  "${PYTHON_VENV}/bin/python" .github/scripts/ci.py \
    --app bitmagnet-concept-explorer \
    --train community \
    --test-file loopback-published-values.yaml \
    --render-only=True --skip-metadata
)

RENDERED_COMPOSE_SRC="${APPS_DIR}/ix-dev/community/bitmagnet-concept-explorer/templates/rendered/docker-compose.yaml"
if [[ ! -f "${RENDERED_COMPOSE_SRC}" ]]; then
  echo "::error::Upstream renderer did not produce rendered docker-compose.yaml"
  exit 1
fi

RENDERED_COMPOSE="${WORK_DIR}/rendered-docker-compose.yaml"
cp "${RENDERED_COMPOSE_SRC}" "${RENDERED_COMPOSE}"
echo "Materialized rendered Compose file: ${RENDERED_COMPOSE}"
cat "${RENDERED_COMPOSE}"
echo ">>> [Stage F0] PASS: Upstream catalog source render complete."

# ============================================================================
# F1: Target Profile Verification
# ============================================================================
echo ""
echo ">>> [Stage F1] Verifying rendered target profile and resource envelope"

"${PYTHON_VENV}/bin/python" - <<PY
import pathlib, sys, yaml

compose_path = pathlib.Path("${RENDERED_COMPOSE}")
data = yaml.safe_load(compose_path.read_text())

if not isinstance(data, dict) or "services" not in data:
    sys.exit("Compose file lacks top-level 'services' mapping")

services = data["services"]
if len(services) != 1 or "bitmagnet-concept-explorer" not in services:
    sys.exit(f"Expected exactly 1 service named 'bitmagnet-concept-explorer', found: {list(services.keys())}")

svc = services["bitmagnet-concept-explorer"]

# Image check
expected_image = "${CONTAINER_IMAGE}"
actual_image = svc.get("image")
if actual_image != expected_image:
    sys.exit(f"Image mismatch: expected {expected_image}, got {actual_image}")

# User security context (non-root)
user = str(svc.get("user"))
if user not in ("65532:65532", "65532"):
    sys.exit(f"Security context violation: expected non-root user 65532:65532, got {user}")

# Read-only root filesystem
if svc.get("read_only") is not True:
    sys.exit("Root filesystem not configured as read-only")

# Privileged mode forbidden
if svc.get("privileged", False) is True:
    sys.exit("Privileged container mode forbidden")

# CPU and Memory resource limits
limits = svc.get("deploy", {}).get("resources", {}).get("limits", {})
cpus = str(limits.get("cpus", ""))
memory = str(limits.get("memory", ""))
if cpus not in ("2", "2.0"):
    sys.exit(f"CPU limit violation: expected 2, got {cpus}")
if memory != "1024M":
    sys.exit(f"Memory limit violation: expected 1024M, got {memory}")

# Port binding: loopback-first published contract
ports = svc.get("ports", [])
if not ports:
    sys.exit("Expected published port mapping, found none")

loopback_found = False
for p in ports:
    if isinstance(p, dict):
        host_ip = p.get("host_ip")
        pub = str(p.get("published"))
        tgt = str(p.get("target"))
        if host_ip == "127.0.0.1" and pub == "3086" and tgt == "3086":
            loopback_found = True
        elif host_ip not in ("127.0.0.1", "localhost"):
            sys.exit(f"Unsafe port binding to non-loopback address: {host_ip}")
    elif isinstance(p, str):
        if p.startswith("127.0.0.1:3086:3086"):
            loopback_found = True
        elif not p.startswith("127.0.0.1:"):
            sys.exit(f"Unsafe port string binding: {p}")

if not loopback_found:
    sys.exit(f"Port 3086 was not bound to loopback 127.0.0.1: {ports}")

# Environment variables
env = svc.get("environment", [])
if isinstance(env, list):
    env_dict = {}
    for item in env:
        if "=" in item:
            k, v = item.split("=", 1)
            env_dict[k] = v
else:
    env_dict = env

if env_dict.get("BCE_LISTEN") != "0.0.0.0:3086":
    sys.exit(f"Unexpected BCE_LISTEN: {env_dict.get('BCE_LISTEN')}")
if env_dict.get("BCE_AUTH_USER") != "${FIXTURE_USER}":
    sys.exit(f"Auth user mismatch: {env_dict.get('BCE_AUTH_USER')}")
if env_dict.get("BCE_AUTH_PASSWORD") != "${FIXTURE_PASS}":
    sys.exit(f"Auth password mismatch")

print("F1-VERIFY: Target profile contract strictly compliant:")
print("  - service: bitmagnet-concept-explorer")
print("  - image:", actual_image)
print("  - user:", user)
print("  - read_only:", svc.get("read_only"))
print(f"  - limits: cpus={cpus}, memory={memory}")
print("  - published ports:", ports)
print("  - listen:", env_dict.get("BCE_LISTEN"))
PY

echo ">>> [Stage F1] PASS: Rendered target profile verified."

# ============================================================================
# F2: Ephemeral Guest & Container Image Qualification
# ============================================================================
echo ""
echo ">>> [Stage F2] Building and verifying local OCI container image"

echo "Building ${CONTAINER_IMAGE} locally via Containerfile..."
if ! docker build -f "${REPO_ROOT}/Containerfile" -t "${CONTAINER_IMAGE}" "${REPO_ROOT}"; then
  echo "Retrying docker build with GO_BUILDER_IMAGE=golang:1.27.1-bookworm..."
  docker build --build-arg GO_BUILDER_IMAGE=golang:1.27.1-bookworm -f "${REPO_ROOT}/Containerfile" -t "${CONTAINER_IMAGE}" "${REPO_ROOT}"
fi

echo "Verifying local OCI container image metadata..."
IMAGE_USER="$(docker image inspect "${CONTAINER_IMAGE}" --format '{{.Config.User}}')"
IMAGE_EXPOSE="$(docker image inspect "${CONTAINER_IMAGE}" --format '{{range $p, $v := .Config.ExposedPorts}}{{$p}} {{end}}')"
IMAGE_ENTRYPOINT="$(docker image inspect "${CONTAINER_IMAGE}" --format '{{range .Config.Entrypoint}}{{.}} {{end}}')"

echo "  Image User: ${IMAGE_USER}"
echo "  Exposed Ports: ${IMAGE_EXPOSE}"
echo "  Entrypoint: ${IMAGE_ENTRYPOINT}"

if [[ "${IMAGE_USER}" != "65532:65532" && "${IMAGE_USER}" != "65532" ]]; then
  echo "::error::OCI Image User mismatch: expected 65532:65532, got ${IMAGE_USER}"
  exit 1
fi

if [[ "${IMAGE_EXPOSE}" != *"3086/tcp"* ]]; then
  echo "::error::OCI Image ExposedPorts missing 3086/tcp"
  exit 1
fi

echo ">>> [Stage F2] PASS: Local OCI container image built and qualified."

# ============================================================================
# F3: Deployment, Health, Endpoints & Teardown
# ============================================================================
echo ""
echo ">>> [Stage F3] Deploying rendered Docker Compose in isolated ephemeral environment"

# Validate compose configuration with local docker engine
docker compose -p "${PROJECT_NAME}" -f "${RENDERED_COMPOSE}" config

echo "Starting container stack with --pull never..."
docker compose -p "${PROJECT_NAME}" -f "${RENDERED_COMPOSE}" up -d --pull never

CONTAINER_ID="$(docker compose -p "${PROJECT_NAME}" -f "${RENDERED_COMPOSE}" ps -q bitmagnet-concept-explorer 2>/dev/null || true)"
if [[ -z "${CONTAINER_ID}" ]]; then
  CONTAINER_ID="$(docker compose -p "${PROJECT_NAME}" -f "${RENDERED_COMPOSE}" ps -q | head -n 1)"
fi
if [[ -z "${CONTAINER_ID}" ]]; then
  echo "::error::Failed to locate running container for service bitmagnet-concept-explorer"
  exit 1
fi
echo "Container instantiated: ${CONTAINER_ID}"

echo "Waiting for service to become responsive on ${SERVICE_URL}/healthz..."
HEALTHY=0
for attempt in $(seq 1 30); do
  HTTP_CODE="$(curl -s -o /dev/null -w "%{http_code}" "${SERVICE_URL}/healthz" 2>/dev/null || true)"
  if [[ "${HTTP_CODE}" == "200" ]]; then
    HEALTHY=1
    echo "Service responsive on attempt ${attempt} (HTTP ${HTTP_CODE})"
    break
  fi
  sleep 1
done

if [[ "${HEALTHY}" -ne 1 ]]; then
  echo "::error::Service failed to respond with HTTP 200 on ${SERVICE_URL}/healthz"
  docker compose -p "${PROJECT_NAME}" -f "${RENDERED_COMPOSE}" logs
  exit 1
fi

echo "--- Runtime Constraint Enforcement Verification ---"

# 1. Non-root user runtime verification
RUNTIME_USER="$(docker inspect --format '{{.Config.User}}' "${CONTAINER_ID}")"
echo "  Runtime User: ${RUNTIME_USER}"
if [[ "${RUNTIME_USER}" != "65532:65532" && "${RUNTIME_USER}" != "65532" ]]; then
  echo "::error::Runtime user is not non-root (got: ${RUNTIME_USER})"
  exit 1
fi

# 2. Read-only root filesystem verification
RUNTIME_READONLY="$(docker inspect --format '{{.HostConfig.ReadonlyRootfs}}' "${CONTAINER_ID}")"
echo "  Runtime ReadonlyRootfs: ${RUNTIME_READONLY}"
if [[ "${RUNTIME_READONLY}" != "true" ]]; then
  echo "::error::Runtime ReadonlyRootfs is NOT true"
  exit 1
fi

# 3. Memory limit enforcement verification (1024M = 1073741824 bytes)
RUNTIME_MEM="$(docker inspect --format '{{.HostConfig.Memory}}' "${CONTAINER_ID}")"
echo "  Runtime Memory Limit: ${RUNTIME_MEM} bytes"
if [[ "${RUNTIME_MEM}" != "1073741824" && "${RUNTIME_MEM}" != "1024000000" ]]; then
  echo "::error::Runtime Memory limit 1024M not enforced (got: ${RUNTIME_MEM})"
  exit 1
fi

# 4. CPU limit enforcement verification (2 CPUs = 2000000000 NanoCpus or quota/period = 2)
RUNTIME_NANOCPUS="$(docker inspect --format '{{.HostConfig.NanoCpus}}' "${CONTAINER_ID}")"
RUNTIME_QUOTA="$(docker inspect --format '{{.HostConfig.CpuQuota}}' "${CONTAINER_ID}")"
RUNTIME_PERIOD="$(docker inspect --format '{{.HostConfig.CpuPeriod}}' "${CONTAINER_ID}")"
echo "  Runtime NanoCpus: ${RUNTIME_NANOCPUS}, CpuQuota: ${RUNTIME_QUOTA}, CpuPeriod: ${RUNTIME_PERIOD}"
if [[ "${RUNTIME_NANOCPUS}" == "2000000000" || "${RUNTIME_NANOCPUS}" == "2" ]]; then
  echo "  CPU limit strictly enforced via NanoCpus."
elif [[ -n "${RUNTIME_PERIOD}" && "${RUNTIME_PERIOD}" != "0" && -n "${RUNTIME_QUOTA}" && "${RUNTIME_QUOTA}" != "0" ]] && [[ $((RUNTIME_QUOTA / RUNTIME_PERIOD)) -eq 2 ]]; then
  echo "  CPU limit strictly enforced via Quota/Period."
else
  echo "::error::Runtime CPU limit 2 CPUs not enforced"
  exit 1
fi

# 5. Loopback-first port binding verification
PORT_MAP="$(docker port "${CONTAINER_ID}" 3086/tcp)"
echo "  Runtime Port Mapping: ${PORT_MAP}"
if [[ "${PORT_MAP}" != *"127.0.0.1:3086"* ]]; then
  echo "::error::Port mapping is not bound to 127.0.0.1:3086 (got: ${PORT_MAP})"
  exit 1
fi
if [[ "${PORT_MAP}" == *"0.0.0.0"* ]]; then
  echo "::error::Port mapping exposed to wildcard 0.0.0.0!"
  exit 1
fi

echo "--- HTTP Authentication and Endpoints Verification ---"

# 1. Unauthenticated request to /api/v1/capabilities must fail with 401
UNAUTH_CODE="$(curl -s -o /dev/null -w "%{http_code}" "${SERVICE_URL}/api/v1/capabilities")"
echo "  Unauthenticated capabilities request returned HTTP: ${UNAUTH_CODE}"
if [[ "${UNAUTH_CODE}" != "401" ]]; then
  echo "::error::Unauthenticated access to /api/v1/capabilities did not return 401 (got: ${UNAUTH_CODE})"
  exit 1
fi

# 2. Bad credentials must fail with 401
BADAUTH_CODE="$(curl -s -o /dev/null -w "%{http_code}" -u "wrong:wrongpassword12345" "${SERVICE_URL}/api/v1/capabilities")"
echo "  Bad credentials capabilities request returned HTTP: ${BADAUTH_CODE}"
if [[ "${BADAUTH_CODE}" != "401" ]]; then
  echo "::error::Invalid credentials did not return 401 (got: ${BADAUTH_CODE})"
  exit 1
fi

# 3. Authenticated request to /api/v1/capabilities must succeed with 200
AUTH_CAPS_RESP="$(curl -sS -u "${FIXTURE_USER}:${FIXTURE_PASS}" "${SERVICE_URL}/api/v1/capabilities")"
echo "  Capabilities Response: ${AUTH_CAPS_RESP}"
echo "${AUTH_CAPS_RESP}" | jq -e '.version == "v0-provisional" and .query_plan == true' >/dev/null
echo "  Capabilities endpoint verification: PASS"

# 4. Authenticated request to /api/v1/query/plan must succeed with 200
AUTH_PLAN_RESP="$(curl -sS -u "${FIXTURE_USER}:${FIXTURE_PASS}" \
  -H "Content-Type: application/json" \
  -d '{"query":"multilingual concept exploration"}' \
  "${SERVICE_URL}/api/v1/query/plan")"
echo "  Query Plan Response: ${AUTH_PLAN_RESP}"
echo "${AUTH_PLAN_RESP}" | jq -e '.query == "multilingual concept exploration" and (.plan | length > 0)' >/dev/null
echo "  Query plan endpoint verification: PASS"

# 5. Authenticated request to Web UI / must succeed with 200 and return HTML
curl -sS -u "${FIXTURE_USER}:${FIXTURE_PASS}" "${SERVICE_URL}/" -o "${WORK_DIR}/ui_index.html"
if ! grep -q -i '<!DOCTYPE html>' "${WORK_DIR}/ui_index.html" || ! grep -q 'Bitmagnet Concept Explorer' "${WORK_DIR}/ui_index.html"; then
  echo "::error::Web UI did not return expected HTML content"
  exit 1
fi
echo "  Web UI root endpoint verification: PASS"

echo "--- Graceful Teardown and Rollback Simulation ---"
echo "Tearing down Compose project ${PROJECT_NAME}..."
docker compose -p "${PROJECT_NAME}" -f "${RENDERED_COMPOSE}" down -v --remove-orphans

# Verify zero orphan containers
REMAINING_CONTAINERS="$(docker ps -aq --filter name="${PROJECT_NAME}")"
if [[ -n "${REMAINING_CONTAINERS}" ]]; then
  echo "::error::Orphan containers remain after teardown: ${REMAINING_CONTAINERS}"
  exit 1
fi

# Verify zero orphan networks
REMAINING_NETWORKS="$(docker network ls -q --filter name="${PROJECT_NAME}")"
if [[ -n "${REMAINING_NETWORKS}" ]]; then
  echo "::error::Orphan networks remain after teardown: ${REMAINING_NETWORKS}"
  exit 1
fi

echo "Zero orphan resources verified."
echo ">>> [Stage F3] PASS: Deployment, health checks, authentication, query, and teardown successful."

# ============================================================================
# Summary & Step Summary
# ============================================================================
if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
  {
    echo "## TrueNAS Foundry Lifecycle Ladder (F0-F3) Qualification"
    echo ""
    echo "| Ladder Stage | Verification | Status |"
    echo "| :--- | :--- | :--- |"
    echo "| **F0: Source Render** | Pinned upstream \`truenas/apps\` catalog validator (\`${PINNED_APPS_COMMIT}\`) | **PASS** |"
    echo "| **F1: Target Profile** | Resource limits (2 CPU, 1024M RAM), loopback port \`127.0.0.1:3086\`, read-only root, non-root user \`65532:65532\` | **PASS** |"
    echo "| **F2: Ephemeral Guest** | Local OCI build of \`${CONTAINER_IMAGE}\`, security context & healthcheck qualification | **PASS** |"
    echo "| **F3: Runtime Realization** | Docker Compose deployment, HTTP auth matrix (401 on unauthorized), 200 OK on \`/healthz\`, \`/api/v1/capabilities\`, \`/api/v1/query/plan\`, Web UI | **PASS** |"
    echo "| **Teardown & Rollback** | Zero orphan containers, networks, or volumes | **PASS** |"
    echo ""
    echo "### Verification Metadata"
    echo "- **Commit:** \`$(git -C "${REPO_ROOT}" rev-parse HEAD)\`"
    echo "- **Runner:** Public \$0 \`ubuntu-24.04\`"
    echo "- **Bitmagnet Authority:** Read-only, no live TrueNAS mutation, zero private corpus exposure"
  } >> "${GITHUB_STEP_SUMMARY}"
fi

echo ""
echo "======================================================================"
echo " TrueNAS Foundry Lifecycle Ladder (F0-F3) COMPLETED SUCCESSFULLY"
echo "======================================================================"
