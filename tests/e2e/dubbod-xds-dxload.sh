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
DUBBOD_XDS_PORT="${DUBBOD_XDS_PORT:-26010}"
DUBBOD_LOCAL_PORT="${DUBBOD_LOCAL_PORT:-26010}"
DXLOAD_DIR="${DXLOAD_DIR:-/Users/mfordjody/laboratory/opensource/dxload}"
DXLOAD_CONFIG="${DXLOAD_CONFIG:-${DXLOAD_DIR}/examples/basic.yaml}"
DXLOAD_DURATION="${DXLOAD_DURATION:-20s}"
DXLOAD_METRICS_PORT="${DXLOAD_METRICS_PORT:-127.0.0.1:0}"

if [[ ! -d "${DXLOAD_DIR}" ]]; then
  echo "dxload directory not found: ${DXLOAD_DIR}" >&2
  exit 1
fi
if [[ ! -f "${DXLOAD_CONFIG}" ]]; then
  echo "dxload config not found: ${DXLOAD_CONFIG}" >&2
  exit 1
fi

"${KUBECTL}" -n "${DUBBOD_NAMESPACE}" rollout status "deploy/${DUBBOD_DEPLOYMENT}" --timeout=120s
"${KUBECTL}" -n "${DUBBOD_NAMESPACE}" get "svc/${DUBBOD_SERVICE}" >/dev/null

pf_log="$(mktemp -t dubbod-xds-port-forward.XXXXXX.log)"
"${KUBECTL}" -n "${DUBBOD_NAMESPACE}" port-forward "svc/${DUBBOD_SERVICE}" \
  "${DUBBOD_LOCAL_PORT}:${DUBBOD_XDS_PORT}" >"${pf_log}" 2>&1 &
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
  if nc -z 127.0.0.1 "${DUBBOD_LOCAL_PORT}" >/dev/null 2>&1; then
    break
  fi
  if ! kill -0 "${pf_pid}" >/dev/null 2>&1; then
    cat "${pf_log}" >&2
    exit 1
  fi
  sleep 0.1
done

if ! nc -z 127.0.0.1 "${DUBBOD_LOCAL_PORT}" >/dev/null 2>&1; then
  cat "${pf_log}" >&2
  echo "dubbod xDS port-forward did not become ready" >&2
  exit 1
fi

(
  cd "${DXLOAD_DIR}"
  go run . cluster \
    --address "127.0.0.1:${DUBBOD_LOCAL_PORT}" \
    --config "${DXLOAD_CONFIG}" \
    --duration "${DXLOAD_DURATION}" \
    --metrics-port "${DXLOAD_METRICS_PORT}" \
    --fail-on-errors=true \
    --require-responses=true
)

"${KUBECTL}" -n "${DUBBOD_NAMESPACE}" logs "deploy/${DUBBOD_DEPLOYMENT}" --since=2m | grep -E "new connection for node|XDS: Pushing|Push Status" >/dev/null
