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

# Regenerates dubbod/gui/resources/data/vendor.js — the console's bundled
# preact + htm runtime. The console must not load anything from a CDN, so the
# runtime is vendored and embedded into the binary alongside app.js.
#
# Usage: tools/gui/vendor.sh   (requires npm)

set -euo pipefail

PREACT_VERSION=10.29.7
HTM_VERSION=3.1.1

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
out="${repo_root}/dubbod/gui/resources/data/vendor.js"
work="$(mktemp -d)"
trap 'rm -rf "${work}"' EXIT

cd "${work}"
npm init -y >/dev/null
npm install --silent --no-audit --no-fund \
  "preact@${PREACT_VERSION}" "htm@${HTM_VERSION}" esbuild >/dev/null

cat > entry.js <<'JS'
export { h, Fragment, render } from "preact";
export { useState, useEffect, useMemo, useRef, useCallback } from "preact/hooks";
export { default as htm } from "htm";
JS

./node_modules/.bin/esbuild entry.js \
  --bundle --format=esm --minify --legal-comments=none --outfile=bundle.js

{
  cat <<EOF
/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/*
 * Bundled third-party runtime for the dubbod console. Vendored so the console
 * has no CDN dependency and works in air-gapped clusters.
 *
 *   preact ${PREACT_VERSION}   (MIT)      https://github.com/preactjs/preact
 *   preact/hooks     (MIT)      https://github.com/preactjs/preact
 *   htm ${HTM_VERSION}        (Apache-2.0) https://github.com/developit/htm
 *
 * Regenerate with tools/gui/vendor.sh — do not edit by hand.
 */
EOF
  cat bundle.js
} > "${out}"

echo "wrote ${out} ($(wc -c < "${out}") bytes)"
