#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

export REPO="${REPO:-dubml}"
export TAG="${TAG:-debug}"
export PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"

exec docker buildx bake -f docker-bake.hcl "$@"
