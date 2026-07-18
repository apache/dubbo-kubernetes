#!/usr/bin/env bash
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# End-to-end smoke test: builds the dubbod image, starts a kind cluster,
# installs the base + dubbod Helm charts, and asserts that the control
# plane serves xDS state for workloads and ServiceEntry configuration.
#
# Requirements: docker, kind, kubectl, helm.
#
# Environment knobs:
#   CLUSTER_NAME    kind cluster name           (default: dubbo-e2e)
#   IMAGE           dubbod image to build/load  (default: kdubbo/dubbod:debug)
#   DUBBOD_REPLICAS control plane replicas      (default: 2, exercises HA)
#   UPGRADE_FROM_VERSION previous release to install before upgrading (default: 0.4.3)
#   UPGRADE_FROM_CHART   local previous chart path; skips release download
#   UPGRADE_FROM_IMAGE   image expected by the previous chart (default: kdubbo/dubbod:debug)
#   SKIP_BUILD      set to 1 to reuse an already-built ${IMAGE}
#   KEEP_CLUSTER    set to 1 to keep the kind cluster after the run

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CLUSTER_NAME="${CLUSTER_NAME:-dubbo-e2e}"
IMAGE="${IMAGE:-kdubbo/dubbod:debug}"
DUBBOD_REPLICAS="${DUBBOD_REPLICAS:-2}"
UPGRADE_FROM_VERSION="${UPGRADE_FROM_VERSION:-0.4.3}"
UPGRADE_FROM_CHART="${UPGRADE_FROM_CHART:-}"
UPGRADE_FROM_IMAGE="${UPGRADE_FROM_IMAGE:-kdubbo/dubbod:debug}"
SYSTEM_NS="dubbo-system"
APP_NS="e2e"
KUBECTL=(kubectl --context "kind-${CLUSTER_NAME}")
UPGRADE_TMP_DIR=""
PREVIOUS_CHART=""

log() { echo "--- $*"; }

fail() {
  echo "FAIL: $*" >&2
  echo "--- diagnostics: pods ---" >&2
  "${KUBECTL[@]}" get pods -A -o wide >&2 || true
  echo "--- diagnostics: dubbod logs ---" >&2
  "${KUBECTL[@]}" -n "${SYSTEM_NS}" logs deploy/dubbod --tail=100 >&2 || true
  exit 1
}

cleanup() {
  if [[ -n "${PF_PID:-}" ]]; then kill "${PF_PID}" 2>/dev/null || true; fi
  if [[ "${KEEP_CLUSTER:-0}" != "1" ]]; then
    kind delete cluster --name "${CLUSTER_NAME}" || true
  fi
  if [[ -n "${UPGRADE_TMP_DIR}" && "${UPGRADE_TMP_DIR}" == */dubbo-upgrade.* ]]; then
    rm -rf -- "${UPGRADE_TMP_DIR}"
  fi
}
trap cleanup EXIT

prepare_previous_chart() {
  if [[ -n "${UPGRADE_FROM_CHART}" ]]; then
    [[ -e "${UPGRADE_FROM_CHART}" ]] || fail "previous chart not found: ${UPGRADE_FROM_CHART}"
    PREVIOUS_CHART="${UPGRADE_FROM_CHART}"
    return
  fi

  UPGRADE_TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/dubbo-upgrade.XXXXXX")"
  local chart_asset="dubbod-${UPGRADE_FROM_VERSION}.tgz"
  local release_url="https://github.com/apache/dubbo-kubernetes/releases/download/${UPGRADE_FROM_VERSION}"
  if curl -fsSL --retry 3 "${release_url}/${chart_asset}" -o "${UPGRADE_TMP_DIR}/${chart_asset}" 2>/dev/null; then
    log "using packaged chart from release ${UPGRADE_FROM_VERSION}"
    curl -fsSL --retry 3 "${release_url}/${chart_asset}.sha256" \
      -o "${UPGRADE_TMP_DIR}/${chart_asset}.sha256"
    (cd "${UPGRADE_TMP_DIR}" && sha256sum -c "${chart_asset}.sha256")
    PREVIOUS_CHART="${UPGRADE_TMP_DIR}/${chart_asset}"
    return
  fi

  log "release ${UPGRADE_FROM_VERSION} predates packaged charts; using its tagged source chart"
  local source_archive="${UPGRADE_TMP_DIR}/source.tar.gz"
  curl -fsSL --retry 3 \
    "https://github.com/apache/dubbo-kubernetes/archive/refs/tags/${UPGRADE_FROM_VERSION}.tar.gz" \
    -o "${source_archive}"
  tar -xzf "${source_archive}" -C "${UPGRADE_TMP_DIR}"
  PREVIOUS_CHART="${UPGRADE_TMP_DIR}/dubbo-kubernetes-${UPGRADE_FROM_VERSION}/manifests/charts/dubbod"
  [[ -f "${PREVIOUS_CHART}/Chart.yaml" ]] || fail "tagged release chart not found: ${PREVIOUS_CHART}"
}

if [[ "${SKIP_BUILD:-0}" != "1" ]]; then
  log "building ${IMAGE}"
  docker build -f "${ROOT}/dubbod/discovery/docker/dockerfile.dubbod" -t "${IMAGE}" "${ROOT}"
fi

if [[ "${IMAGE}" != "${UPGRADE_FROM_IMAGE}" ]]; then
  log "tagging ${IMAGE} as ${UPGRADE_FROM_IMAGE} for the previous chart"
  docker tag "${IMAGE}" "${UPGRADE_FROM_IMAGE}"
fi

if ! kind get clusters 2>/dev/null | grep -qx "${CLUSTER_NAME}"; then
  log "creating kind cluster ${CLUSTER_NAME}"
  kind create cluster --name "${CLUSTER_NAME}" --wait 120s
fi

log "loading ${IMAGE} into kind"
kind load docker-image "${IMAGE}" --name "${CLUSTER_NAME}"
if [[ "${IMAGE}" != "${UPGRADE_FROM_IMAGE}" ]]; then
  kind load docker-image "${UPGRADE_FROM_IMAGE}" --name "${CLUSTER_NAME}"
fi

log "installing Gateway API CRDs"
# Pin to the sigs.k8s.io/gateway-api version in go.mod.
GATEWAY_API_VERSION="${GATEWAY_API_VERSION:-v1.4.1}"
"${KUBECTL[@]}" apply -f "https://github.com/kubernetes-sigs/gateway-api/releases/download/${GATEWAY_API_VERSION}/standard-install.yaml"

log "installing base chart (CRDs)"
helm upgrade --install dubbo-base "${ROOT}/manifests/charts/base" \
  --kube-context "kind-${CLUSTER_NAME}" \
  -n "${SYSTEM_NS}" --create-namespace

install_dubbod() {
  local chart="$1"
  local image="$2"
  # The CNI daemonset needs privileged host access; keep the smoke test to the
  # control plane itself.
  helm upgrade --install dubbod "${chart}" \
    --kube-context "kind-${CLUSTER_NAME}" \
    -n "${SYSTEM_NS}" \
    --set global.proxyless.cni.enabled=false \
    --set-string global.proxyless.cni.image="${image}" \
    --set replicaCount="${DUBBOD_REPLICAS}"
}

prepare_previous_chart

log "installing dubbod ${UPGRADE_FROM_VERSION} chart (${DUBBOD_REPLICAS} replicas)"
install_dubbod "${PREVIOUS_CHART}" "${UPGRADE_FROM_IMAGE}"

log "waiting for previous dubbod rollout"
"${KUBECTL[@]}" -n "${SYSTEM_NS}" rollout status deploy/dubbod --timeout=300s \
  || fail "dubbod ${UPGRADE_FROM_VERSION} deployment did not become ready"

log "upgrading dubbod ${UPGRADE_FROM_VERSION} to the current chart"
install_dubbod "${ROOT}/manifests/charts/dubbod" "${IMAGE}" \
  || fail "upgrade from dubbod ${UPGRADE_FROM_VERSION} to the current chart failed"
"${KUBECTL[@]}" -n "${SYSTEM_NS}" rollout status deploy/dubbod --timeout=300s \
  || fail "upgraded dubbod deployment did not become ready"

HELM_REVISION="$(helm history dubbod --kube-context "kind-${CLUSTER_NAME}" -n "${SYSTEM_NS}" | awk 'END {print $1}')"
[[ "${HELM_REVISION}" -ge 2 ]] || fail "helm release revision is ${HELM_REVISION}, want at least 2 after upgrade"
DEPLOYED_IMAGE="$("${KUBECTL[@]}" -n "${SYSTEM_NS}" get deploy dubbod -o jsonpath='{.spec.template.spec.containers[0].image}')"
[[ "${DEPLOYED_IMAGE}" == "${IMAGE}" ]] \
  || fail "upgraded deployment image is ${DEPLOYED_IMAGE}, want ${IMAGE}"

if [[ "${DUBBOD_REPLICAS}" -gt 1 ]]; then
  log "asserting PodDisruptionBudget exists for HA"
  "${KUBECTL[@]}" -n "${SYSTEM_NS}" get pdb dubbod >/dev/null \
    || fail "PodDisruptionBudget dubbod not found with replicaCount=${DUBBOD_REPLICAS}"
fi

# Regression: re-running helm upgrade against a live control plane must not
# hit server-side apply conflicts on the fields dubbod manages at runtime
# (webhook caBundle / failurePolicy).
log "re-running helm upgrade against the live control plane"
install_dubbod "${ROOT}/manifests/charts/dubbod" "${IMAGE}" \
  || fail "helm upgrade over a running dubbod failed (SSA field conflict?)"

log "deploying sample workload (httpbin)"
"${KUBECTL[@]}" create namespace "${APP_NS}" --dry-run=client -o yaml | "${KUBECTL[@]}" apply -f -
"${KUBECTL[@]}" -n "${APP_NS}" apply -f "${ROOT}/samples/httpbin/httpbin.yaml"
"${KUBECTL[@]}" -n "${APP_NS}" rollout status deploy/httpbin --timeout=300s \
  || fail "httpbin deployment did not become ready"

log "applying ServiceEntry through the validating webhook"
"${KUBECTL[@]}" -n "${APP_NS}" apply -f "${ROOT}/tests/e2e/testdata/serviceentry.yaml" \
  || fail "valid ServiceEntry was rejected"

log "port-forwarding dubbod monitoring port"
"${KUBECTL[@]}" -n "${SYSTEM_NS}" port-forward deploy/dubbod 18080:8080 >/dev/null 2>&1 &
PF_PID=$!

probe() { curl -sf --max-time 5 "http://127.0.0.1:18080$1"; }

# The registry and config propagate asynchronously; retry before failing.
retry() {
  local desc="$1"; shift
  for _ in $(seq 1 30); do
    if "$@" >/dev/null 2>&1; then return 0; fi
    sleep 2
  done
  fail "timed out waiting for: ${desc}"
}

check_registry_service() { probe /debug/registryz | grep -q "httpbin.${APP_NS}.svc"; }
check_registry_serviceentry() { probe /debug/registryz | grep -q "external.example.com"; }
check_metrics() { probe /metrics | grep -q "^dubbod_"; }

retry "monitoring endpoint up" probe /version
log "asserting /metrics exposes dubbod metrics"
retry "dubbod metrics" check_metrics
log "asserting httpbin service is in the registry"
retry "httpbin in /debug/registryz" check_registry_service
log "asserting ServiceEntry host is in the registry"
retry "ServiceEntry in /debug/registryz" check_registry_serviceentry

log "e2e smoke test passed"
