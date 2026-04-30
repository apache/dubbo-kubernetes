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
DUBBOD_NAMESPACE="${DUBBOD_NAMESPACE:-dubbo-system}"
DUBBOD_SERVICE="${DUBBOD_SERVICE:-dubbod}"
DUBBOD_DEPLOYMENT="${DUBBOD_DEPLOYMENT:-dubbod}"
ADDONS_DIR="${ADDONS_DIR:-samples/addons}"
PROMETHEUS_SERVICE="${PROMETHEUS_SERVICE:-prometheus}"
GRAFANA_SERVICE="${GRAFANA_SERVICE:-grafana}"
PROMETHEUS_LOCAL_PORT="${PROMETHEUS_LOCAL_PORT:-19090}"
PROMETHEUS_QUERY="${PROMETHEUS_QUERY:-up{job=\"dubbod\"}}"
DRY_RUN_ONLY="${DUBBO_OBSERVABILITY_DRY_RUN_ONLY:-false}"

if [[ ! -d "${ADDONS_DIR}" ]]; then
  echo "addons directory not found: ${ADDONS_DIR}" >&2
  exit 1
fi

"${KUBECTL}" apply --dry-run=client -f "${ADDONS_DIR}" >/dev/null

if [[ "${DRY_RUN_ONLY}" == "true" ]]; then
  exit 0
fi

"${KUBECTL}" -n "${DUBBOD_NAMESPACE}" rollout status "deploy/${DUBBOD_DEPLOYMENT}" --timeout=180s
"${KUBECTL}" apply -f "${ADDONS_DIR}"
"${KUBECTL}" -n "${DUBBOD_NAMESPACE}" rollout status "deploy/${PROMETHEUS_SERVICE}" --timeout=180s
"${KUBECTL}" -n "${DUBBOD_NAMESPACE}" rollout status "deploy/${GRAFANA_SERVICE}" --timeout=180s
"${KUBECTL}" -n "${DUBBOD_NAMESPACE}" get "svc/${PROMETHEUS_SERVICE}" "svc/${GRAFANA_SERVICE}" >/dev/null

scrape_annotation="$("${KUBECTL}" -n "${DUBBOD_NAMESPACE}" get "svc/${DUBBOD_SERVICE}" -o jsonpath='{.metadata.annotations.prometheus\.io/scrape}')"
metrics_port="$("${KUBECTL}" -n "${DUBBOD_NAMESPACE}" get "svc/${DUBBOD_SERVICE}" -o jsonpath='{.metadata.annotations.prometheus\.io/port}')"
if [[ "${scrape_annotation}" != "true" || "${metrics_port}" != "8080" ]]; then
  echo "dubbod service prometheus annotations invalid: scrape=${scrape_annotation} port=${metrics_port}" >&2
  exit 1
fi

pf_log="$(mktemp -t dubbod-observability-prometheus.XXXXXX.log)"
"${KUBECTL}" -n "${DUBBOD_NAMESPACE}" port-forward "svc/${PROMETHEUS_SERVICE}" \
  "${PROMETHEUS_LOCAL_PORT}:9090" >"${pf_log}" 2>&1 &
pf_pid="$!"

cleanup() {
  if kill -0 "${pf_pid}" >/dev/null 2>&1; then
    kill "${pf_pid}" >/dev/null 2>&1 || true
    wait "${pf_pid}" >/dev/null 2>&1 || true
  fi
  rm -f "${pf_log}"
}
trap cleanup EXIT

for _ in $(seq 1 100); do
  if nc -z 127.0.0.1 "${PROMETHEUS_LOCAL_PORT}" >/dev/null 2>&1; then
    break
  fi
  if ! kill -0 "${pf_pid}" >/dev/null 2>&1; then
    cat "${pf_log}" >&2
    exit 1
  fi
  sleep 0.1
done

if ! nc -z 127.0.0.1 "${PROMETHEUS_LOCAL_PORT}" >/dev/null 2>&1; then
  cat "${pf_log}" >&2
  echo "prometheus port-forward did not become ready" >&2
  exit 1
fi

python3 - "${PROMETHEUS_LOCAL_PORT}" "${PROMETHEUS_QUERY}" <<'PY'
import json
import sys
import urllib.parse
import urllib.request

port, query = sys.argv[1], sys.argv[2]
url = f"http://127.0.0.1:{port}/api/v1/query?" + urllib.parse.urlencode({"query": query})
with urllib.request.urlopen(url, timeout=10) as response:
    payload = json.load(response)

if payload.get("status") != "success":
    raise SystemExit(f"prometheus query failed: {payload}")
if not payload.get("data", {}).get("result"):
    raise SystemExit(f"prometheus query returned no series: {query}")
PY
