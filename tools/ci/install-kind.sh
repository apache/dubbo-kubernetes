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

version="${1:-}"
if [[ ! "${version}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "usage: $0 vMAJOR.MINOR.PATCH" >&2
  exit 2
fi

case "$(uname -m)" in
  x86_64) arch="amd64" ;;
  aarch64 | arm64) arch="arm64" ;;
  *)
    echo "unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

if [[ "$(uname -s)" != "Linux" ]]; then
  echo "unsupported operating system: $(uname -s)" >&2
  exit 1
fi

binary="kind-linux-${arch}"
download_base="https://github.com/kubernetes-sigs/kind/releases/download/${version}"
install_root="${RUNNER_TEMP:-${TMPDIR:-/tmp}}/dubbo-ci-tools"
work_dir="$(mktemp -d "${TMPDIR:-/tmp}/dubbo-kind.XXXXXX")"

cleanup() {
  rm -rf -- "${work_dir}"
}
trap cleanup EXIT

curl -fsSLo "${work_dir}/${binary}" "${download_base}/${binary}"
curl -fsSLo "${work_dir}/${binary}.sha256sum" "${download_base}/${binary}.sha256sum"
expected="$(awk 'NR == 1 {print $1}' "${work_dir}/${binary}.sha256sum")"
actual="$(sha256sum "${work_dir}/${binary}" | awk '{print $1}')"
if [[ -z "${expected}" || "${actual}" != "${expected}" ]]; then
  echo "kind checksum verification failed for ${binary}" >&2
  exit 1
fi

mkdir -p "${install_root}/bin"
install -m 0755 "${work_dir}/${binary}" "${install_root}/bin/kind"

if [[ -n "${GITHUB_PATH:-}" ]]; then
  echo "${install_root}/bin" >> "${GITHUB_PATH}"
fi

"${install_root}/bin/kind" version
