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

WD=$(dirname "$0")
ROOT=$(cd "${WD}/.."; pwd)

HUB="${HUB:-docker.io/dubml}"
TAG="${TAG:-latest}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"
PUSH="${PUSH:-false}"

# Read base images from tools/docker.yaml
awk '
  /- name:/ { name=$3 }
  /dockerfile:/ { df=$2 }
  /base: true/ { print name, df }
' "${ROOT}/tools/docker.yaml" | while read -r name dockerfile; do
  echo "Building base image: ${name} from ${dockerfile}..."
  IMAGE="${HUB}/${name}:${TAG}"
  BUILD_ARGS=(
    -f "${ROOT}/${dockerfile}"
    -t "${IMAGE}"
  )
  if [[ "${PUSH}" == "true" ]]; then
    BUILD_ARGS+=(--platform "${PLATFORMS}" --push)
  else
    if [[ "${PLATFORMS}" == *","* ]]; then
      echo "Note: Multi-platform build without PUSH=true; building for host platform with --load."
      BUILD_ARGS+=(--load)
    else
      BUILD_ARGS+=(--platform "${PLATFORMS}" --load)
    fi
  fi
  docker buildx build "${BUILD_ARGS[@]}" "${ROOT}"
  echo "Successfully built base image: ${IMAGE}"
done
