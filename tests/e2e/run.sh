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
# Requirements: docker, kind, kubectl, helm, jq.
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
#   KIND            path to the kind binary      (default: kind)
#   KIND_NODE_IMAGE kind node image override     (default: kind release default)
#   ACTIVATION_E2E  install KEDA and run real scale-to-zero E2E (default: 0)
#   AI_MESH_E2E     run no-key HTTP/LLM/MCP/A2A E2E (default: 0)
#   DXGATE_IMAGE    prebuilt dxgate image used by managed Gateways
#   KEDA_VERSION    pinned KEDA chart/app version (default: 2.20.2)

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
KIND="${KIND:-kind}"
KIND_NODE_IMAGE="${KIND_NODE_IMAGE:-}"
ACTIVATION_E2E="${ACTIVATION_E2E:-0}"
AI_MESH_E2E="${AI_MESH_E2E:-0}"
DXGATE_IMAGE="${DXGATE_IMAGE:-kdubbo/dxgate:latest}"
AGENT_MOCK_IMAGE="${AGENT_MOCK_IMAGE:-kdubbo/agent-mock:latest}"
ACTIVATION_APP_IMAGE="${ACTIVATION_APP_IMAGE:-kdubbo/activation-e2e:latest}"
ACTIVATION_CLIENT_IMAGE="${ACTIVATION_CLIENT_IMAGE:-kdubbo/activation-client:latest}"
KEDA_VERSION="${KEDA_VERSION:-2.20.2}"

log() { echo "--- $*"; }

fail() {
  echo "FAIL: $*" >&2
  echo "--- diagnostics: pods ---" >&2
  "${KUBECTL[@]}" get pods -A -o wide >&2 || true
  echo "--- diagnostics: dubbod logs ---" >&2
  "${KUBECTL[@]}" -n "${SYSTEM_NS}" logs deploy/dubbod --tail=100 >&2 || true
  echo "--- diagnostics: managed gateway logs ---" >&2
  local pod
  while read -r pod; do
    [[ -n "${pod}" ]] || continue
    echo "--- ${APP_NS}/${pod} current ---" >&2
    "${KUBECTL[@]}" -n "${APP_NS}" logs "${pod}" --all-containers --tail=100 >&2 || true
    echo "--- ${APP_NS}/${pod} previous ---" >&2
    "${KUBECTL[@]}" -n "${APP_NS}" logs "${pod}" --all-containers --previous --tail=100 >&2 || true
  done < <("${KUBECTL[@]}" -n "${APP_NS}" get pods \
    -l gateway.networking.k8s.io/gateway-name \
    -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null || true)
  exit 1
}

cleanup() {
  if [[ -n "${PF_PID:-}" ]]; then kill "${PF_PID}" 2>/dev/null || true; fi
  if [[ -n "${AI_PF_PID:-}" ]]; then kill "${AI_PF_PID}" 2>/dev/null || true; fi
  if [[ "${KEEP_CLUSTER:-0}" != "1" ]]; then
    "${KIND}" delete cluster --name "${CLUSTER_NAME}" || true
  fi
  if [[ -n "${UPGRADE_TMP_DIR}" && "${UPGRADE_TMP_DIR}" == */dubbo-upgrade.* ]]; then
    rm -rf -- "${UPGRADE_TMP_DIR}"
  fi
}
trap cleanup EXIT

apply_activation_fixture() {
  sed \
    -e "s#kdubbo/activation-e2e:latest#${ACTIVATION_APP_IMAGE}#g" \
    -e "s#kdubbo/activation-client:latest#${ACTIVATION_CLIENT_IMAGE}#g" \
    "$1" \
    | "${KUBECTL[@]}" apply -f -
}

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

if ! "${KIND}" get clusters 2>/dev/null | grep -qx "${CLUSTER_NAME}"; then
  log "creating kind cluster ${CLUSTER_NAME}"
  KIND_CREATE_ARGS=(create cluster --name "${CLUSTER_NAME}" --wait 120s)
  if [[ -n "${KIND_NODE_IMAGE}" ]]; then
    KIND_CREATE_ARGS+=(--image "${KIND_NODE_IMAGE}")
  fi
  "${KIND}" "${KIND_CREATE_ARGS[@]}"
fi

# A kept cluster is useful for debugging and repeated activation runs. Reset
# release-scoped state so runtime-mutated webhook fields cannot conflict with
# the next Helm install after a previous namespace was deleted.
helm uninstall dubbod --kube-context "kind-${CLUSTER_NAME}" -n "${SYSTEM_NS}" --ignore-not-found >/dev/null 2>&1 || true
helm uninstall dubbo-base --kube-context "kind-${CLUSTER_NAME}" -n "${SYSTEM_NS}" --ignore-not-found >/dev/null 2>&1 || true
"${KUBECTL[@]}" delete namespace "${APP_NS}" "${SYSTEM_NS}" --ignore-not-found --wait=true >/dev/null
"${KUBECTL[@]}" delete validatingwebhookconfiguration,mutatingwebhookconfiguration \
  -l app=dubbod --ignore-not-found >/dev/null

log "loading ${IMAGE} into kind"
"${KIND}" load docker-image "${IMAGE}" --name "${CLUSTER_NAME}"
if [[ "${IMAGE}" != "${UPGRADE_FROM_IMAGE}" ]]; then
  "${KIND}" load docker-image "${UPGRADE_FROM_IMAGE}" --name "${CLUSTER_NAME}"
fi
if [[ "${ACTIVATION_E2E}" == "1" || "${AI_MESH_E2E}" == "1" ]]; then
  docker image inspect "${DXGATE_IMAGE}" >/dev/null 2>&1 \
    || fail "data-plane E2E requires prebuilt ${DXGATE_IMAGE}"
fi
if [[ "${ACTIVATION_E2E}" == "1" ]]; then
  if [[ "${SKIP_BUILD:-0}" != "1" ]]; then
    log "building ${ACTIVATION_APP_IMAGE}"
    docker build -t "${ACTIVATION_APP_IMAGE}" "${ROOT}/tests/e2e/activationapp"
  else
    docker image inspect "${ACTIVATION_APP_IMAGE}" >/dev/null 2>&1 \
      || fail "SKIP_BUILD=1 requires prebuilt ${ACTIVATION_APP_IMAGE}"
  fi
  docker tag "${IMAGE}" "${ACTIVATION_CLIENT_IMAGE}"
  log "loading activation data-plane images into kind"
  "${KIND}" load docker-image \
    "${DXGATE_IMAGE}" \
    "${ACTIVATION_APP_IMAGE}" \
    "${ACTIVATION_CLIENT_IMAGE}" \
    --name "${CLUSTER_NAME}"
fi
if [[ "${AI_MESH_E2E}" == "1" ]]; then
  if [[ "${SKIP_BUILD:-0}" != "1" ]]; then
    log "building ${AGENT_MOCK_IMAGE}"
    docker build -t "${AGENT_MOCK_IMAGE}" "${ROOT}/tests/e2e/agentmock"
  else
    docker image inspect "${AGENT_MOCK_IMAGE}" >/dev/null 2>&1 \
      || fail "SKIP_BUILD=1 requires prebuilt ${AGENT_MOCK_IMAGE}"
  fi
  log "loading AI mesh data-plane images into kind"
  "${KIND}" load docker-image \
    "${DXGATE_IMAGE}" \
    "${AGENT_MOCK_IMAGE}" \
    --name "${CLUSTER_NAME}"
fi

log "installing Gateway API CRDs"
# Pin to the sigs.k8s.io/gateway-api version in go.mod. HTTPRoute retry is an
# Extended feature carried by the experimental CRD bundle.
GATEWAY_API_VERSION="${GATEWAY_API_VERSION:-v1.4.1}"
"${KUBECTL[@]}" apply --server-side -f "https://github.com/kubernetes-sigs/gateway-api/releases/download/${GATEWAY_API_VERSION}/experimental-install.yaml"

if [[ "${ACTIVATION_E2E}" == "1" ]]; then
  log "installing KEDA ${KEDA_VERSION}"
  helm repo add kedacore https://kedacore.github.io/charts --force-update
  helm repo update kedacore
  helm upgrade --install keda kedacore/keda \
    --version "${KEDA_VERSION}" \
    --kube-context "kind-${CLUSTER_NAME}" \
    -n keda --create-namespace
  "${KUBECTL[@]}" -n keda rollout status deploy/keda-operator --timeout=300s \
    || fail "KEDA operator did not become ready"
  "${KUBECTL[@]}" -n keda rollout status deploy/keda-admission-webhooks --timeout=300s \
    || fail "KEDA admission webhook did not become ready"
fi

log "installing base chart (CRDs)"
helm upgrade --install dubbo-base "${ROOT}/manifests/charts/base" \
  --kube-context "kind-${CLUSTER_NAME}" \
  -n "${SYSTEM_NS}" --create-namespace

install_dubbod() {
  local chart="$1"
  local image="$2"
  local -a chart_values
  if [[ "${chart}" == "${ROOT}/manifests/charts/dubbod" ]]; then
    chart_values=(
      --set-string "image=${image}"
      --set-string "gateway.image=${DXGATE_IMAGE}"
    )
  else
    # The previous chart still exposes its legacy CNI configuration.
    chart_values=(
      --set "global.proxyless.cni.enabled=false"
      --set-string "global.proxyless.cni.image=${image}"
      --set-string "global.gateway.image=${DXGATE_IMAGE}"
    )
  fi
  helm upgrade --install dubbod "${chart}" \
    --kube-context "kind-${CLUSTER_NAME}" \
    -n "${SYSTEM_NS}" \
    "${chart_values[@]}" \
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

log "registering a VM workload through WorkloadEntry"
"${KUBECTL[@]}" -n "${APP_NS}" apply -f "${ROOT}/tests/e2e/testdata/vm-workload.yaml" \
  || fail "valid VM WorkloadEntry was rejected"

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
check_registry_vm() { probe /debug/registryz | grep -q "reviews-vm.mesh.local"; }
check_vm_health() {
  local health="$1"
  probe /debug/endpointz | tr -d '\n ' \
    | grep -q "\"hostname\":\"reviews-vm.mesh.local\".*\"address\":\"192.0.2.10\".*\"health\":\"${health}\".*\"network\":\"vm-network\".*\"locality\":\"us-east-1/zone-a/rack-1\".*\"weight\":7"
}
check_metrics() { probe /metrics | grep -q "^dubbod_"; }

retry "monitoring endpoint up" probe /version
log "asserting /metrics exposes dubbod metrics"
retry "dubbod metrics" check_metrics
log "asserting httpbin service is in the registry"
retry "httpbin in /debug/registryz" check_registry_service
log "asserting ServiceEntry host is in the registry"
retry "ServiceEntry in /debug/registryz" check_registry_serviceentry
log "asserting VM service and endpoint topology are published"
retry "VM ServiceEntry in /debug/registryz" check_registry_vm
retry "healthy VM endpoint in /debug/endpointz" check_vm_health HEALTHY

log "marking the VM endpoint unhealthy through the status subresource"
"${KUBECTL[@]}" -n "${APP_NS}" patch workloadentry reviews-vm --subresource=status --type=merge \
  -p '{"status":{"conditions":[{"type":"Ready","status":"False","reason":"E2EHealthCheck"}]}}' \
  || fail "could not update VM health status"
retry "unhealthy VM endpoint in /debug/endpointz" check_vm_health UNHEALTHY

log "restoring the VM endpoint health"
"${KUBECTL[@]}" -n "${APP_NS}" patch workloadentry reviews-vm --subresource=status --type=merge \
  -p '{"status":{"conditions":[{"type":"Ready","status":"True","reason":"E2EHealthCheck"}]}}' \
  || fail "could not restore VM health status"
retry "recovered VM endpoint in /debug/endpointz" check_vm_health HEALTHY

# --- On-demand activation -----------------------------------------------
#
# Everything here failed silently in ways unit tests cannot see: a CRD that is
# registered in Go but missing from the chart, a headless Service that is not
# actually headless, or a gateway that never learns where to report. Each is a
# working control plane that simply never activates anything.

log "asserting the activation CRD is installed"
"${KUBECTL[@]}" get crd serviceactivationpolicies.networking.dubbo.apache.org >/dev/null \
  || fail "ServiceActivationPolicy CRD is missing; the chart's CRD bundle is out of sync with the Go schema"

log "asserting both activation Services exist, one of them headless"
"${KUBECTL[@]}" -n "${SYSTEM_NS}" get svc dubbod-activation >/dev/null \
  || fail "dubbod-activation Service not found"
ACTIVATION_HEADLESS_IP="$("${KUBECTL[@]}" -n "${SYSTEM_NS}" get svc dubbod-activation-replicas -o jsonpath='{.spec.clusterIP}')"
# A VIP here would silently break scale-up: a gateway would report to whichever
# replica the VIP picked, and KEDA polls a replica chosen independently.
[[ "${ACTIVATION_HEADLESS_IP}" == "None" ]] \
  || fail "dubbod-activation-replicas has clusterIP ${ACTIVATION_HEADLESS_IP}, want None (headless)"
ACTIVATION_NOT_READY="$("${KUBECTL[@]}" -n "${SYSTEM_NS}" get svc dubbod-activation-replicas -o jsonpath='{.spec.publishNotReadyAddresses}')"
[[ "${ACTIVATION_NOT_READY}" == "true" ]] \
  || fail "dubbod-activation-replicas does not publish not-ready addresses; a starting replica would miss reports"

log "applying a ServiceActivationPolicy through the validating webhook"
"${KUBECTL[@]}" -n "${APP_NS}" apply -f "${ROOT}/tests/e2e/testdata/activation-policy.yaml" \
  || fail "valid ServiceActivationPolicy was rejected"

# The controller is what turns a policy into scaler state; if it is not running
# the policy is inert and nothing else in this section would notice.
check_policy_accepted() {
  "${KUBECTL[@]}" -n "${APP_NS}" get serviceactivationpolicy httpbin \
    -o jsonpath='{.status.conditions[?(@.type=="Accepted")].status}' | grep -qx True
}
log "asserting the policy controller writes status back"
retry "Accepted condition on the policy" check_policy_accepted

log "asserting a managed gateway is told where to report demand"
"${KUBECTL[@]}" -n "${APP_NS}" apply -f "${ROOT}/tests/e2e/testdata/gateway.yaml" \
  || fail "Gateway was rejected"
check_gateway_deployment() { "${KUBECTL[@]}" -n "${APP_NS}" get deploy public-dubbo >/dev/null; }
retry "managed gateway deployment" check_gateway_deployment
GATEWAY_ENV="$("${KUBECTL[@]}" -n "${APP_NS}" get deploy public-dubbo \
  -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="DXGATE_ACTIVATION_CONTROL_PLANE")].value}')"
[[ "${GATEWAY_ENV}" == dubbod-activation-replicas.* ]] \
  || fail "gateway reports to '${GATEWAY_ENV}', want the headless activation Service"
# Reports are attributed per reporter; without a distinct identity two gateway
# replicas overwrite each other's counts instead of adding to them.
"${KUBECTL[@]}" -n "${APP_NS}" get deploy public-dubbo \
  -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="POD_NAME")].valueFrom.fieldRef.fieldPath}' \
  | grep -qx metadata.name \
  || fail "gateway does not inject POD_NAME; demand reports would not be attributable to a replica"

if [[ "${AI_MESH_E2E}" == "1" ]]; then
  log "applying mesh-native HTTP, OpenAI, Anthropic, MCP, and A2A sample"
  sed "s#kdubbo/agent-mock:latest#${AGENT_MOCK_IMAGE}#g" \
    "${ROOT}/samples/ai-mesh/backends.yaml" \
    | "${KUBECTL[@]}" -n "${APP_NS}" apply -f -
  "${KUBECTL[@]}" -n "${APP_NS}" apply -f "${ROOT}/samples/ai-mesh/services.yaml"
  "${KUBECTL[@]}" -n "${APP_NS}" apply -f "${ROOT}/samples/ai-mesh/routes.yaml"
  "${KUBECTL[@]}" -n "${APP_NS}" rollout status deploy/agent-mock --timeout=300s \
    || fail "agent mock deployment did not become ready"
  "${KUBECTL[@]}" -n "${APP_NS}" rollout status deploy/public-dubbo --timeout=300s \
    || fail "mesh gateway deployment did not become ready"

  "${KUBECTL[@]}" get crd dxgateservices.networking.dubbo.apache.org >/dev/null \
    || fail "DxgateService CRD is missing"
  "${KUBECTL[@]}" -n "${APP_NS}" get role public-dubbo-credentials >/dev/null \
    || fail "dxgate credential Role is missing"
  "${KUBECTL[@]}" -n "${APP_NS}" get rolebinding public-dubbo-credentials >/dev/null \
    || fail "dxgate credential RoleBinding is missing"
  "${KUBECTL[@]}" auth can-i get secret/agent-credentials \
    --as="system:serviceaccount:${APP_NS}:public-dubbo" -n "${APP_NS}" \
    | grep -qx yes || fail "dxgate ServiceAccount cannot resolve referenced Secret"

  log "port-forwarding mesh gateway"
  "${KUBECTL[@]}" -n "${APP_NS}" port-forward svc/public-dubbo 18081:80 >/dev/null 2>&1 &
  AI_PF_PID=$!
  ai_get() { curl -sf --max-time 10 "http://127.0.0.1:18081$1"; }
  ai_post() {
    local path="$1" body="$2"
    curl -sf --max-time 10 -H 'content-type: application/json' \
      -d "${body}" "http://127.0.0.1:18081${path}"
  }
  ai_openai() {
    curl -sf --max-time 10 -H 'content-type: application/json' \
      -H 'x-client-key: mock-client-key' \
      -d '{"model":"gpt-mock","messages":[{"role":"user","content":"ping"}]}' \
      http://127.0.0.1:18081/openai/chat/completions
  }

  retry "ordinary /users Service route" ai_get /users
  [[ "$(ai_get /users | jq -r .path)" == "/users" ]] \
    || fail "ordinary /users route returned the wrong backend response"
  [[ "$(ai_get /orders | jq -r .path)" == "/orders" ]] \
    || fail "ordinary /orders route returned the wrong backend response"
  retry "OpenAI DxgateService route and Secret resolution" ai_openai
  OPENAI_RESPONSE="$(ai_openai)"
  [[ "$(jq -r .choices[0].message.content <<<"${OPENAI_RESPONSE}")" == "openai-mock" ]] \
    || fail "OpenAI mock response was not proxied"
  [[ "$(jq -r .provider_authorization <<<"${OPENAI_RESPONSE}")" == "Bearer mock-openai-key" ]] \
    || fail "OpenAI provider credential was not resolved from the Secret"

  ANTHROPIC_RESPONSE="$(ai_post /anthropic/chat/completions \
    '{"model":"claude-mock","messages":[{"role":"user","content":"ping"}]}')"
  [[ "$(jq -r .choices[0].message.content <<<"${ANTHROPIC_RESPONSE}")" == "anthropic-mock" ]] \
    || fail "Anthropic native response was not translated to OpenAI"

  MCP_RESPONSE="$(ai_post /mcp '{"jsonrpc":"2.0","id":1,"method":"tools/list"}')"
  [[ "$(jq -r '[.result.tools[].name] | sort | join(",")' <<<"${MCP_RESPONSE}")" == "calendar,search" ]] \
    || fail "MCP tools/list was not federated across both targets"
  [[ "$(ai_post /mcp '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"search","arguments":{}}}' \
      | jq -r .result.content[0].text)" == "search-ok" ]] \
    || fail "MCP tools/call did not reach the selected target"

  [[ "$(ai_get /.well-known/agent-card.json | jq -r .name)" == "planner" ]] \
    || fail "A2A Agent Card was not proxied"
  [[ "$(ai_post /a2a '{"jsonrpc":"2.0","id":3,"method":"message/send","params":{}}' \
      | jq -r .result.status.state)" == "completed" ]] \
    || fail "A2A task request did not complete"

  check_agent_config_programmed() {
    local pod
    pod="$("${KUBECTL[@]}" -n "${APP_NS}" get pods \
      -l gateway.networking.k8s.io/gateway-name=public \
      -o jsonpath='{.items[0].metadata.name}')"
    "${KUBECTL[@]}" get --raw \
      "/api/v1/namespaces/${APP_NS}/pods/${pod}:26021/proxy/debug/config" \
      | jq -e '
          ([.providers[].name] | length) == 2 and
          ([.backends[] | select(.type == "llm")] | length) == 2 and
          ([.backends[] | select(.type == "mcp")] | length) == 2 and
          ([.backends[] | select(.type == "a2a")] | length) == 1 and
          ([.routes[] | .protocol] | sort | join(",")) == "a2a,llm,llm,mcp"
        ' >/dev/null
  }
  retry "compiled AgentConfig visible in dxgate /debug/config" check_agent_config_programmed
  log "mesh-native HTTP, LLM, MCP, and A2A E2E passed"
fi

if [[ "${ACTIVATION_E2E}" == "1" ]]; then
  log "asserting multiple Gateways have isolated resources"
  "${KUBECTL[@]}" -n "${APP_NS}" get deploy dxgate-gateway public-dubbo >/dev/null \
    || fail "canonical and public Gateway deployments do not coexist"
  "${KUBECTL[@]}" -n "${APP_NS}" rollout status deploy/dxgate-gateway --timeout=300s \
    || fail "canonical Activator gateway did not become ready"
  "${KUBECTL[@]}" -n "${APP_NS}" rollout status deploy/public-dubbo --timeout=300s \
    || fail "second managed gateway did not become ready"

  log "deploying the proxyless activation target and KEDA ScaledObject"
  apply_activation_fixture "${ROOT}/tests/e2e/testdata/eastwest-activation.yaml"
  "${KUBECTL[@]}" apply -f "${ROOT}/tests/e2e/testdata/eastwest-activation-scaledobject.yaml"
  "${KUBECTL[@]}" -n "${APP_NS}" wait --for=condition=Ready scaledobject/payment --timeout=180s \
    || fail "KEDA ScaledObject did not become ready"

  check_all_activators_have_payment_route() {
    local pod pods
    pods="$("${KUBECTL[@]}" -n "${APP_NS}" get pods \
      -l gateway.networking.k8s.io/gateway-name=dxgate-gateway \
      --field-selector=status.phase=Running \
      -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}')"
    [[ "$(wc -w <<<"${pods}")" -ge 2 ]] || return 1
    while read -r pod; do
      "${KUBECTL[@]}" get --raw \
        "/api/v1/namespaces/${APP_NS}/pods/${pod}:26021/proxy/debug/config" \
        | jq -e '.listeners[]
          | select(.bind == "0.0.0.0:15080")
          | .virtual_hosts[]
          | select(.name == "activation|payment.e2e.svc.cluster.local|8080")' \
        >/dev/null \
        || return 1
    done <<<"${pods}"
  }
  retry "all Activator replicas receive the payment activation route" \
    check_all_activators_have_payment_route

  check_payment_policy_runtime_ready() {
    local conditions
    conditions="$("${KUBECTL[@]}" -n "${APP_NS}" get serviceactivationpolicy payment \
      -o jsonpath='{range .status.conditions[*]}{.type}={.status}{"\n"}{end}')"
    grep -qx 'ScalerReady=True' <<<"${conditions}" &&
      grep -qx 'ActivatorReady=True' <<<"${conditions}"
  }
  retry "payment policy scaler and Activator readiness" \
    check_payment_policy_runtime_ready

  check_payment_scaled_to_zero() {
    [[ "$("${KUBECTL[@]}" -n "${APP_NS}" get deploy payment -o jsonpath='{.spec.replicas}')" == "0" ]]
  }
  retry "KEDA scale payment to zero" check_payment_scaled_to_zero

  log "removing one control-plane and one Activator replica before cold demand"
  "${KUBECTL[@]}" -n "${SYSTEM_NS}" delete pod \
    "$("${KUBECTL[@]}" -n "${SYSTEM_NS}" get pod -l app=dubbod -o jsonpath='{.items[0].metadata.name}')" \
    --wait=false
  "${KUBECTL[@]}" -n "${APP_NS}" delete pod \
    "$("${KUBECTL[@]}" -n "${APP_NS}" get pod \
      -l gateway.networking.k8s.io/gateway-name=dxgate-gateway \
      -o jsonpath='{.items[0].metadata.name}')" \
    --wait=false

  "${KUBECTL[@]}" -n "${SYSTEM_NS}" rollout status deploy/dubbod --timeout=180s \
    || fail "control-plane replica did not recover"
  "${KUBECTL[@]}" -n "${APP_NS}" rollout status deploy/dxgate-gateway --timeout=180s \
    || fail "Activator replica did not recover"
  retry "all Activator replicas retain the payment route after failover" \
    check_all_activators_have_payment_route
  retry "payment policy remains runtime-ready after HA failover" \
    check_payment_policy_runtime_ready

  log "sending one proxyless request while payment is at zero"
  "${KUBECTL[@]}" -n "${APP_NS}" delete pod payment-client --ignore-not-found
  apply_activation_fixture "${ROOT}/tests/e2e/testdata/eastwest-activation-client.yaml"
  activation_metrics() {
    local pod
    while read -r pod; do
      "${KUBECTL[@]}" get --raw \
        "/api/v1/namespaces/${APP_NS}/pods/${pod}:26021/proxy/metrics"
    done < <("${KUBECTL[@]}" -n "${APP_NS}" get pods \
      -l gateway.networking.k8s.io/gateway-name=dxgate-gateway \
      -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}')
  }
  activation_payment_requests_total() {
    activation_metrics \
      | awk '/^dxgate_http_route_requests_total\{/ && /cluster="outbound\|8080\|\|payment\.e2e\.svc\.cluster\.local"/ { total += $NF } END { print total + 0 }'
  }
  # held_requests is an instantaneous gauge and can return to zero before a
  # polling assertion observes it. The durable proof is the complete sequence:
  # replicas were zero above, KEDA now scales from pending demand, the request
  # succeeds, and the Activator route counter increases.
  check_payment_scaled_from_zero() {
    [[ "$("${KUBECTL[@]}" -n "${APP_NS}" get deploy payment -o jsonpath='{.spec.replicas}')" -ge 1 ]]
  }
  retry "KEDA scale payment from zero" check_payment_scaled_from_zero
  log "KEDA scaled payment from zero"
  "${KUBECTL[@]}" -n "${APP_NS}" rollout status deploy/payment --timeout=180s \
    || fail "payment did not become ready after KEDA activation"
  "${KUBECTL[@]}" -n "${APP_NS}" wait \
    --for=jsonpath='{.status.containerStatuses[?(@.name=="client")].state.terminated.exitCode}'=0 \
    pod/payment-client --timeout=120s \
    || fail "held proxyless request did not complete after automatic scale-up"
  check_payment_client_success() {
    "${KUBECTL[@]}" -n "${APP_NS}" logs payment-client -c client 2>/dev/null | grep -qx payment-ok
  }
  retry "cold proxyless response is available in the pod log" \
    check_payment_client_success
  COLD_ACTIVATOR_REQUESTS="$(activation_payment_requests_total)"
  log "cold request completed through Activator (route requests=${COLD_ACTIVATOR_REQUESTS})"
  [[ "${COLD_ACTIVATOR_REQUESTS}" -ge 1 ]] \
    || fail "cold request completed without an Activator data-plane metric"

  log "asserting EDS converges back to a direct hot path"
  # The first ready endpoint and its EDS update are independent events. Keep
  # one backend alive while polling the data plane, otherwise the short test
  # cooldown can scale it back to zero before xDS convergence is observable.
  "${KUBECTL[@]}" -n "${APP_NS}" patch scaledobject payment --type=merge \
    -p '{"spec":{"minReplicaCount":1}}' >/dev/null
  hot_request_bypasses_activator() {
    local before after
    before="$(activation_payment_requests_total)"
    "${KUBECTL[@]}" -n "${APP_NS}" delete pod payment-client --ignore-not-found --wait=true >/dev/null
    apply_activation_fixture "${ROOT}/tests/e2e/testdata/eastwest-activation-client.yaml" >/dev/null
    "${KUBECTL[@]}" -n "${APP_NS}" wait \
      --for=jsonpath='{.status.containerStatuses[?(@.name=="client")].state.terminated.exitCode}'=0 \
      pod/payment-client --timeout=30s >/dev/null || return 1
    check_payment_client_success || return 1
    after="$(activation_payment_requests_total)"
    [[ "${after}" == "${before}" ]]
  }
  retry "hot proxyless request bypasses the Activator after EDS convergence" \
    hot_request_bypasses_activator
  log "hot proxyless request bypassed the Activator"

  "${KUBECTL[@]}" -n "${APP_NS}" patch scaledobject payment --type=merge \
    -p '{"spec":{"minReplicaCount":0}}' >/dev/null
  retry "KEDA scale payment back to zero" check_payment_scaled_to_zero
  log "real KEDA zero-to-one-to-zero activation passed"
fi

log "e2e smoke test passed"
