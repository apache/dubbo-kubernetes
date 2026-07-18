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
#   SKIP_BUILD      set to 1 to reuse an already-built ${IMAGE}
#   KEEP_CLUSTER    set to 1 to keep the kind cluster after the run

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CLUSTER_NAME="${CLUSTER_NAME:-dubbo-e2e}"
IMAGE="${IMAGE:-kdubbo/dubbod:debug}"
DUBBOD_REPLICAS="${DUBBOD_REPLICAS:-2}"
SYSTEM_NS="dubbo-system"
APP_NS="e2e"
KUBECTL=(kubectl --context "kind-${CLUSTER_NAME}")

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
}
trap cleanup EXIT

if [[ "${SKIP_BUILD:-0}" != "1" ]]; then
  log "building ${IMAGE}"
  docker build -f "${ROOT}/dubbod/discovery/docker/dockerfile.dubbod" -t "${IMAGE}" "${ROOT}"
fi

if ! kind get clusters 2>/dev/null | grep -qx "${CLUSTER_NAME}"; then
  log "creating kind cluster ${CLUSTER_NAME}"
  kind create cluster --name "${CLUSTER_NAME}" --wait 120s
fi

log "loading ${IMAGE} into kind"
kind load docker-image "${IMAGE}" --name "${CLUSTER_NAME}"

log "installing Gateway API CRDs"
# Pin to the sigs.k8s.io/gateway-api version in go.mod.
GATEWAY_API_VERSION="${GATEWAY_API_VERSION:-v1.4.1}"
"${KUBECTL[@]}" apply -f "https://github.com/kubernetes-sigs/gateway-api/releases/download/${GATEWAY_API_VERSION}/standard-install.yaml"

log "installing base chart (CRDs)"
helm upgrade --install dubbo-base "${ROOT}/manifests/charts/base" \
  --kube-context "kind-${CLUSTER_NAME}" \
  -n "${SYSTEM_NS}" --create-namespace

install_dubbod() {
  # The CNI daemonset needs privileged host access; keep the smoke test to the
  # control plane itself.
  helm upgrade --install dubbod "${ROOT}/manifests/charts/dubbod" \
    --kube-context "kind-${CLUSTER_NAME}" \
    -n "${SYSTEM_NS}" \
    --set global.proxyless.cni.enabled=false \
    --set replicaCount="${DUBBOD_REPLICAS}"
}

log "installing dubbod chart (${DUBBOD_REPLICAS} replicas)"
install_dubbod

log "waiting for dubbod rollout"
"${KUBECTL[@]}" -n "${SYSTEM_NS}" rollout status deploy/dubbod --timeout=300s \
  || fail "dubbod deployment did not become ready"

if [[ "${DUBBOD_REPLICAS}" -gt 1 ]]; then
  log "asserting PodDisruptionBudget exists for HA"
  "${KUBECTL[@]}" -n "${SYSTEM_NS}" get pdb dubbod >/dev/null \
    || fail "PodDisruptionBudget dubbod not found with replicaCount=${DUBBOD_REPLICAS}"
fi

# Regression: re-running helm upgrade against a live control plane must not
# hit server-side apply conflicts on the fields dubbod manages at runtime
# (webhook caBundle / failurePolicy).
log "re-running helm upgrade against the live control plane"
install_dubbod || fail "helm upgrade over a running dubbod failed (SSA field conflict?)"

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
