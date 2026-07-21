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

HUB="${HUB:-kdubbo}"
TAG="${TAG:-latest}"
PUSH="${PUSH:-false}"
# 目标节点架构，与集群节点不一致时必须显式指定，例如 PLATFORM=linux/amd64。
PLATFORM="${PLATFORM:-}"

cd "$(dirname "${BASH_SOURCE[0]}")/src"

build_args=()
if [[ -n "${PLATFORM}" ]]; then
  build_args+=(--platform "${PLATFORM}")
fi

for svc in moviepage details ratings reviews-v1 reviews-v2 reviews-v3; do
  image="${HUB}/moviereview-${svc}:${TAG}"
  echo "==> building ${image}"
  docker build "${build_args[@]}" -t "${image}" "${svc}"
  if [[ "${PUSH}" == "true" ]]; then
    docker push "${image}"
  fi
done
