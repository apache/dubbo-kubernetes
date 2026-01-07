#!/usr/bin/env bash
# shellcheck disable=SC2034

set -e

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
REPO_ROOT="$(dirname "$(dirname "${SCRIPT_DIR}")")"

LOCAL_ARCH=$(uname -m)

if [[ ${TARGET_ARCH} ]]; then
    :
elif [[ ${LOCAL_ARCH} == x86_64 ]]; then
    TARGET_ARCH=amd64
elif [[ ${LOCAL_ARCH} == armv8* ]]; then
    TARGET_ARCH=arm64
elif [[ ${LOCAL_ARCH} == arm64* ]]; then
    TARGET_ARCH=arm64
elif [[ ${LOCAL_ARCH} == aarch64* ]]; then
    TARGET_ARCH=arm64
else
    echo "This system's architecture, ${LOCAL_ARCH}, isn't supported"
    exit 1
fi

LOCAL_OS=$(uname)

# Pass environment set target operating-system to build system
if [[ ${TARGET_OS} ]]; then
    # Target explicitly set
    :
elif [[ $LOCAL_OS == Linux ]]; then
    TARGET_OS=linux
    readlink_flags="-f"
elif [[ $LOCAL_OS == Darwin ]]; then
    TARGET_OS=darwin
    readlink_flags=""
else
    echo "This system's OS, $LOCAL_OS, isn't supported"
    exit 1
fi

TIMEZONE=$(readlink "$readlink_flags" /etc/localtime | sed -e 's/^.*zoneinfo\///')

ENV_BLOCKLIST="${ENV_BLOCKLIST:-^_\|^PATH=\|^GOPATH=\|^GOROOT=\|^SHELL=\|^EDITOR=\|^TMUX=\|^USER=\|^HOME=\|^PWD=\|^TERM=\|^RUBY_\|^GEM_\|^rvm_\|^SSH=\|^TMPDIR=\|^CC=\|^CXX=\|^MAKEFILE_LIST=}"

# Build image to use
TOOLS_REGISTRY_PROVIDER=${TOOLS_REGISTRY_PROVIDER:-docker.io}
PROJECT_ID=${PROJECT_ID:-mfordjody}
if [[ "${IMAGE_VERSION:-}" == "" ]]; then
  IMAGE_VERSION=master
fi
if [[ "${IMAGE_NAME:-}" == "" ]]; then
  IMAGE_NAME=build-tools
fi

#TOOLS_REGISTRY_PROVIDER=${TOOLS_REGISTRY_PROVIDER:-gcr.io}
#PROJECT_ID=${PROJECT_ID:-istio-testing}
#if [[ "${IMAGE_VERSION:-}" == "" ]]; then
#  IMAGE_VERSION=master-dd350f492cf194be812d6f79d13e450f10b62e94
#fi
#if [[ "${IMAGE_NAME:-}" == "" ]]; then
#  IMAGE_NAME=build-tools
#fi

CONTAINER_CLI="${CONTAINER_CLI:-docker}"

# Try to use the latest cached image we have. Use at your own risk, may have incompatibly-old versions
if [[ "${LATEST_CACHED_IMAGE:-}" != "" ]]; then
  prefix="$(<<<"$IMAGE_VERSION" cut -d- -f1)"
  query="${TOOLS_REGISTRY_PROVIDER}/${PROJECT_ID}/${IMAGE_NAME}:${prefix}-*"
  latest="$("${CONTAINER_CLI}" images --filter=reference="${query}" --format "{{.CreatedAt|json}}~{{.Repository}}:{{.Tag}}~{{.CreatedSince}}" | sort -n -r | head -n1)"
  IMG="$(<<<"$latest" cut -d~ -f2)"
  if [[ "${IMG}" == "" ]]; then
    echo "Attempted to use LATEST_CACHED_IMAGE, but found no images matching ${query}" >&2
    exit 1
  fi
  echo "Using cached image $IMG, created $(<<<"$latest" cut -d~ -f3)" >&2
fi

IMG="${IMG:-${TOOLS_REGISTRY_PROVIDER}/${PROJECT_ID}/${IMAGE_NAME}:${IMAGE_VERSION}}"

TARGET_OUT="${TARGET_OUT:-$(pwd)/out/${TARGET_OS}_${TARGET_ARCH}}"
TARGET_OUT_LINUX="${TARGET_OUT_LINUX:-$(pwd)/out/linux_${TARGET_ARCH}}"

CONTAINER_TARGET_OUT="${CONTAINER_TARGET_OUT:-/work/out/${TARGET_OS}_${TARGET_ARCH}}"
CONTAINER_TARGET_OUT_LINUX="${CONTAINER_TARGET_OUT_LINUX:-/work/out/linux_${TARGET_ARCH}}"

BUILD_WITH_CONTAINER=0

# LOCAL_OUT should point to architecture where we are currently running versus the desired.
# This is used when we need to run a build artifact during tests or later as part of another
# target.
if [[ "${FOR_BUILD_CONTAINER:-0}" -eq "1" ]]; then
    # Override variables with container specific
    TARGET_OUT=${CONTAINER_TARGET_OUT}
    TARGET_OUT_LINUX=${CONTAINER_TARGET_OUT_LINUX}
    REPO_ROOT=/work
    LOCAL_OUT="${TARGET_OUT_LINUX}"
else
    LOCAL_OUT="${TARGET_OUT}"
fi

go_os_arch=${LOCAL_OUT##*/}
# Golang OS/Arch format
LOCAL_GO_OS=${go_os_arch%_*}
LOCAL_GO_ARCH=${go_os_arch##*_}

VARS=(
      CONTAINER_TARGET_OUT
      CONTAINER_TARGET_OUT_LINUX
      TARGET_OUT
      TARGET_OUT_LINUX
      LOCAL_GO_OS
      LOCAL_GO_ARCH
      LOCAL_OUT
      LOCAL_OS
      TARGET_OS
      LOCAL_ARCH
      TARGET_ARCH
      TIMEZONE
      CONTAINER_CLI
      IMG
      IMAGE_NAME
      IMAGE_VERSION
      REPO_ROOT
      BUILD_WITH_CONTAINER
)

# For non container build, we need to write env to file
if [[ "${1}" == "envfile" ]]; then
    # ! does a variable-variable https://stackoverflow.com/a/10757531/374797
    for var in "${VARS[@]}"; do
        echo "${var}"="${!var}"
    done
else
    for var in "${VARS[@]}"; do
        # shellcheck disable=SC2163
        export "${var}"
    done
fi
