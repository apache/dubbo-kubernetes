#!/usr/bin/env python3
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

import json
from pathlib import Path
import subprocess
import sys


# buildpacks/pack still imports github.com/docker/docker/volume/mounts. The
# advisories below affect Docker daemon endpoints, but govulncheck links their
# unrelated init helpers to pack's mount parser. Keep this list narrow so a
# future trace into an actual vulnerable Docker symbol still fails the scan.
IGNORED_DOCKER_FINDINGS = {
    "GO-2026-4883",
    "GO-2026-4887",
    "GO-2026-5617",
    "GO-2026-5668",
    "GO-2026-5746",
}

IGNORED_DOCKER_SYMBOLS = {
    ("github.com/docker/docker/api/types/mount", "init"),
    ("github.com/docker/docker/api/types/filters", "init"),
    ("github.com/docker/docker/api/types/registry", "init"),
    ("github.com/docker/docker/api/types/versions", "init"),
    ("github.com/docker/docker/internal/lazyregexp", "FindStringSubmatch"),
    ("github.com/docker/docker/internal/lazyregexp", "MatchString"),
    ("github.com/docker/docker/internal/lazyregexp", "New"),
    ("github.com/docker/docker/internal/lazyregexp", "SubexpNames"),
    ("github.com/docker/docker/internal/lazyregexp", "build"),
    ("github.com/docker/docker/internal/lazyregexp", "init"),
    ("github.com/docker/docker/internal/safepath", "init"),
    ("github.com/docker/docker/internal/unix_noeintr", "init"),
    ("github.com/docker/docker/pkg/homedir", "init"),
    ("github.com/docker/docker/pkg/idtools", "init"),
    ("github.com/docker/docker/pkg/stringid", "init"),
    ("github.com/docker/docker/registry", "init"),
    ("github.com/docker/docker/volume", "init"),
    ("github.com/docker/docker/volume/mounts", "Error"),
    ("github.com/docker/docker/volume/mounts", "NewLCOWParser"),
    ("github.com/docker/docker/volume/mounts", "NewLinuxParser"),
    ("github.com/docker/docker/volume/mounts", "NewWindowsParser"),
    ("github.com/docker/docker/volume/mounts", "ParseMountRaw"),
    ("github.com/docker/docker/volume/mounts", "init"),
}


def is_symbol_finding(finding):
    return any(frame.get("function") for frame in finding.get("trace", []))


def is_ignored_docker_finding(finding):
    if finding.get("osv") not in IGNORED_DOCKER_FINDINGS:
        return False
    if finding.get("fixed_version"):
        return False

    trace = finding.get("trace", [])
    if not trace:
        return False
    vulnerable = trace[0]
    if vulnerable.get("module") != "github.com/docker/docker":
        return False
    return (vulnerable.get("package"), vulnerable.get("function")) in IGNORED_DOCKER_SYMBOLS


def format_trace(finding):
    frames = []
    for frame in finding.get("trace", []):
        package = frame.get("package") or frame.get("module") or "unknown"
        function = frame.get("function")
        frames.append(f"{package}.{function}" if function else package)
    return " -> ".join(frames)


def main():
    patterns = sys.argv[1:] or ["./..."]
    repository = Path(__file__).resolve().parents[1]
    try:
        process = subprocess.run(
            ["govulncheck", "-format=json", *patterns],
            capture_output=True,
            cwd=repository,
            text=True,
        )
    except FileNotFoundError:
        print("govulncheck executable not found", file=sys.stderr)
        return 1

    findings = []
    decoder = json.JSONDecoder()
    position = 0
    try:
        while position < len(process.stdout):
            while position < len(process.stdout) and process.stdout[position].isspace():
                position += 1
            if position >= len(process.stdout):
                break
            event, position = decoder.raw_decode(process.stdout, position)
            finding = event.get("finding")
            if finding and is_symbol_finding(finding):
                findings.append(finding)
    except json.JSONDecodeError as error:
        print(f"invalid govulncheck JSON: {error}", file=sys.stderr)
        print(process.stderr, file=sys.stderr)
        return 1

    if process.returncode not in (0, 3):
        print(process.stderr, file=sys.stderr)
        print(f"govulncheck failed with exit code {process.returncode}", file=sys.stderr)
        return process.returncode
    if process.returncode == 3 and not findings:
        print("govulncheck reported vulnerabilities but no symbol findings were parsed", file=sys.stderr)
        return 3

    ignored = {finding["osv"] for finding in findings if is_ignored_docker_finding(finding)}
    unexpected = [finding for finding in findings if not is_ignored_docker_finding(finding)]

    for osv in sorted(ignored):
        print(f"IGNORED {osv}: pack mount-parser trace does not invoke the vulnerable Docker daemon endpoint")

    if unexpected:
        print("FAIL: actionable vulnerabilities found", file=sys.stderr)
        seen = set()
        for finding in unexpected:
            key = (finding["osv"], format_trace(finding))
            if key in seen:
                continue
            seen.add(key)
            print(f"{finding['osv']}: {key[1]}", file=sys.stderr)
        return 3

    print("PASS: no actionable vulnerabilities found")
    return 0


if __name__ == "__main__":
    sys.exit(main())
