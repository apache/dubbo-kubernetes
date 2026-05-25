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

set -euo pipefail

KUBECTL="${KUBECTL:-kubectl}"
DUBBOCTL="${DUBBOCTL:-go run ./dubboctl}"
PRIMARY_CONTEXT="${PRIMARY_CONTEXT:?PRIMARY_CONTEXT is required}"
REMOTE_CONTEXT="${REMOTE_CONTEXT:?REMOTE_CONTEXT is required}"
PRIMARY_KUBECONFIG="${PRIMARY_KUBECONFIG:-${KUBECONFIG:-${HOME}/.kube/config}}"
REMOTE_CLUSTER_NAME="${REMOTE_CLUSTER_NAME:?REMOTE_CLUSTER_NAME is required}"
REMOTE_WEBHOOK_URL="${REMOTE_WEBHOOK_URL:?REMOTE_WEBHOOK_URL is required}"
REMOTE_XDS_ADDRESS="${REMOTE_XDS_ADDRESS:?REMOTE_XDS_ADDRESS is required}"
REMOTE_CA_ADDRESS="${REMOTE_CA_ADDRESS:?REMOTE_CA_ADDRESS is required}"
CA_BUNDLE_FILE="${CA_BUNDLE_FILE:?CA_BUNDLE_FILE is required}"
REMOTE_KUBECONFIG="${REMOTE_KUBECONFIG:-${KUBECONFIG:-${HOME}/.kube/config}}"
MULTICLUSTER_NETWORK_MODE="${MULTICLUSTER_NETWORK_MODE:-cross-network}"
PRIMARY_CLUSTER_NAME="${PRIMARY_CLUSTER_NAME:-Kubernetes}"
EASTWEST_GATEWAY_PORT="${EASTWEST_GATEWAY_PORT:-15443}"
EASTWEST_SERVICE_TYPE="${EASTWEST_SERVICE_TYPE:-NodePort}"
EASTWEST_NODE_PORT="${EASTWEST_NODE_PORT:-32443}"
REMOTE_EASTWEST_XDS_ADDRESS="${REMOTE_EASTWEST_XDS_ADDRESS:-}"

"${KUBECTL}" --kubeconfig "${PRIMARY_KUBECONFIG}" --context "${PRIMARY_CONTEXT}" cluster-info >/dev/null
"${KUBECTL}" --kubeconfig "${REMOTE_KUBECONFIG}" --context "${REMOTE_CONTEXT}" cluster-info >/dev/null

if [[ "${MULTICLUSTER_NETWORK_MODE}" == "same-network" ]]; then
  primary_pod_cidrs="$("${KUBECTL}" --kubeconfig "${PRIMARY_KUBECONFIG}" --context "${PRIMARY_CONTEXT}" get nodes \
    -o jsonpath='{range .items[*]}{.spec.podCIDR}{"\n"}{end}' | sort -u)"
  remote_pod_cidrs="$("${KUBECTL}" --kubeconfig "${REMOTE_KUBECONFIG}" --context "${REMOTE_CONTEXT}" get nodes \
    -o jsonpath='{range .items[*]}{.spec.podCIDR}{"\n"}{end}' | sort -u)"
  if comm -12 <(printf '%s\n' "${primary_pod_cidrs}") <(printf '%s\n' "${remote_pod_cidrs}") | grep -q .; then
    echo "primary and remote pod CIDRs overlap; same-network multicluster requires disjoint, routable Pod CIDRs" >&2
    exit 1
  fi
else
  primary_gateways="$("${KUBECTL}" --kubeconfig "${PRIMARY_KUBECONFIG}" --context "${PRIMARY_CONTEXT}" -n dubbo-system get deploy dubbod \
    -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="DUBBO_EASTWEST_GATEWAYS")].value}')"
  if [[ "${primary_gateways}" != *"${REMOTE_CLUSTER_NAME}="* || "${primary_gateways}" != *"${PRIMARY_CLUSTER_NAME}="* ]]; then
    echo "dubbod DUBBO_EASTWEST_GATEWAYS=${primary_gateways}; cross-network requires ${REMOTE_CLUSTER_NAME}=... and ${PRIMARY_CLUSTER_NAME}=..." >&2
    exit 1
  fi
  if [[ -z "${REMOTE_EASTWEST_XDS_ADDRESS}" ]]; then
    echo "REMOTE_EASTWEST_XDS_ADDRESS is required for cross-network remote dxgate bootstrap" >&2
    exit 1
  fi
fi

${DUBBOCTL} multicluster create-remote-secret \
  --cluster-name "${REMOTE_CLUSTER_NAME}" \
  --kubeconfig "${REMOTE_KUBECONFIG}" \
  --context "${REMOTE_CONTEXT}" | "${KUBECTL}" --kubeconfig "${PRIMARY_KUBECONFIG}" --context "${PRIMARY_CONTEXT}" apply -f -

${DUBBOCTL} multicluster generate-remote-manifest \
  --cluster-name "${REMOTE_CLUSTER_NAME}" \
  --webhook-url "${REMOTE_WEBHOOK_URL}" \
  --xds-address "${REMOTE_XDS_ADDRESS}" \
  --ca-address "${REMOTE_CA_ADDRESS}" \
  --ca-bundle-file "${CA_BUNDLE_FILE}" | "${KUBECTL}" --kubeconfig "${REMOTE_KUBECONFIG}" --context "${REMOTE_CONTEXT}" apply -f -

if [[ "${MULTICLUSTER_NETWORK_MODE}" != "same-network" ]]; then
  gateway_args=(
    multicluster generate-eastwest-gateway
    --xds-address "${REMOTE_EASTWEST_XDS_ADDRESS}"
    --service-type "${EASTWEST_SERVICE_TYPE}"
    --port "${EASTWEST_GATEWAY_PORT}"
  )
  if [[ "${EASTWEST_NODE_PORT}" != "0" ]]; then
    gateway_args+=(--node-port "${EASTWEST_NODE_PORT}")
  fi
  ${DUBBOCTL} "${gateway_args[@]}" | "${KUBECTL}" --kubeconfig "${REMOTE_KUBECONFIG}" --context "${REMOTE_CONTEXT}" apply -f -
fi

"${KUBECTL}" --kubeconfig "${PRIMARY_KUBECONFIG}" --context "${PRIMARY_CONTEXT}" create namespace app --dry-run=client -o yaml | \
  "${KUBECTL}" --kubeconfig "${PRIMARY_KUBECONFIG}" --context "${PRIMARY_CONTEXT}" apply -f -
"${KUBECTL}" --kubeconfig "${PRIMARY_KUBECONFIG}" --context "${PRIMARY_CONTEXT}" label namespace app dubbo-injection=enabled --overwrite
"${KUBECTL}" --kubeconfig "${PRIMARY_KUBECONFIG}" --context "${PRIMARY_CONTEXT}" apply -f samples/app/deployment.yaml

"${KUBECTL}" --kubeconfig "${REMOTE_KUBECONFIG}" --context "${REMOTE_CONTEXT}" create namespace app --dry-run=client -o yaml | \
  "${KUBECTL}" --kubeconfig "${REMOTE_KUBECONFIG}" --context "${REMOTE_CONTEXT}" apply -f -
"${KUBECTL}" --kubeconfig "${REMOTE_KUBECONFIG}" --context "${REMOTE_CONTEXT}" label namespace app dubbo-injection=enabled --overwrite
"${KUBECTL}" --kubeconfig "${REMOTE_KUBECONFIG}" --context "${REMOTE_CONTEXT}" apply -f samples/app/deployment.yaml

"${KUBECTL}" --kubeconfig "${PRIMARY_KUBECONFIG}" --context "${PRIMARY_CONTEXT}" -n app scale deploy/nginx-v2 --replicas=0
"${KUBECTL}" --kubeconfig "${REMOTE_KUBECONFIG}" --context "${REMOTE_CONTEXT}" -n app scale deploy/nginx-v1 --replicas=0

"${KUBECTL}" --kubeconfig "${PRIMARY_KUBECONFIG}" --context "${PRIMARY_CONTEXT}" -n app rollout status deploy/nginx-v1 --timeout=180s
"${KUBECTL}" --kubeconfig "${PRIMARY_KUBECONFIG}" --context "${PRIMARY_CONTEXT}" -n app rollout status deploy/nginx-v2 --timeout=180s
"${KUBECTL}" --kubeconfig "${PRIMARY_KUBECONFIG}" --context "${PRIMARY_CONTEXT}" -n app rollout status deploy/nginx-consumer --timeout=180s
"${KUBECTL}" --kubeconfig "${REMOTE_KUBECONFIG}" --context "${REMOTE_CONTEXT}" -n app rollout status deploy/nginx-v1 --timeout=180s
"${KUBECTL}" --kubeconfig "${REMOTE_KUBECONFIG}" --context "${REMOTE_CONTEXT}" -n app rollout status deploy/nginx-v2 --timeout=180s
"${KUBECTL}" --kubeconfig "${REMOTE_KUBECONFIG}" --context "${REMOTE_CONTEXT}" -n app rollout status deploy/nginx-consumer --timeout=180s

"${KUBECTL}" --kubeconfig "${PRIMARY_KUBECONFIG}" --context "${PRIMARY_CONTEXT}" apply -f samples/app/meshservice.yaml
if [[ "${MULTICLUSTER_NETWORK_MODE}" != "same-network" ]]; then
  "${KUBECTL}" --kubeconfig "${PRIMARY_KUBECONFIG}" --context "${PRIMARY_CONTEXT}" apply -f samples/multicluster/eastwest-nginx-httproute.yaml
fi

remote_cluster_id="$("${KUBECTL}" --kubeconfig "${REMOTE_KUBECONFIG}" --context "${REMOTE_CONTEXT}" -n app get pod -l app=nginx-consumer \
  -o jsonpath='{.items[0].spec.containers[0].env[?(@.name=="DUBBO_META_CLUSTER_ID")].value}')"
if [[ "${remote_cluster_id}" != "${REMOTE_CLUSTER_NAME}" ]]; then
  echo "remote consumer cluster id = ${remote_cluster_id}, want ${REMOTE_CLUSTER_NAME}" >&2
  exit 1
fi

"${KUBECTL}" --kubeconfig "${PRIMARY_KUBECONFIG}" --context "${PRIMARY_CONTEXT}" -n app exec deploy/nginx-consumer -- \
  dubbod xclient --expect v1=50,v2=50 100 | sort | uniq -c
"${KUBECTL}" --kubeconfig "${REMOTE_KUBECONFIG}" --context "${REMOTE_CONTEXT}" -n app exec deploy/nginx-consumer -- \
  dubbod xclient --expect v1=50,v2=50 100 | sort | uniq -c
