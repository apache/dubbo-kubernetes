#!/usr/bin/env bash

set -e

WD=$(dirname "$0")
WD=$(cd "$WD"; pwd)

export FOR_BUILD_CONTAINER=1
# shellcheck disable=SC1090,SC1091
source "${WD}/setup_env.sh"


MOUNT_SOURCE="${MOUNT_SOURCE:-${PWD}}"
MOUNT_DEST="${MOUNT_DEST:-/work}"

read -ra DOCKER_RUN_OPTIONS <<< "${DOCKER_RUN_OPTIONS:-}"

[[ -t 0 ]] && DOCKER_RUN_OPTIONS+=("-it")
[[ ${UID} -ne 0 ]] && DOCKER_RUN_OPTIONS+=(-u "${UID}:${DOCKER_GID}")

# $CONTAINER_OPTIONS becomes an empty arg when quoted, so SC2086 is disabled for the
# following command only
# shellcheck disable=SC2086
"${CONTAINER_CLI}" run \
    --rm \
    "${DOCKER_RUN_OPTIONS[@]}" \
    --init \
    --sig-proxy=true \
    --cap-add=SYS_ADMIN \
    ${DOCKER_SOCKET_MOUNT:--v /var/run/docker.sock:/var/run/docker.sock} \
    -e DOCKER_HOST=${DOCKER_SOCKET_HOST:-unix:///var/run/docker.sock} \
    $CONTAINER_OPTIONS \
    --env-file <(env | grep -v "${ENV_BLOCKLIST:-^$}") \
    -e IN_BUILD_CONTAINER=1 \
    -e TZ="${TIMEZONE:-$TZ}" \
    --mount "type=bind,source=${MOUNT_SOURCE},destination=/work" \
    --mount "type=volume,source=go,destination=/go" \
    --mount "type=volume,source=gocache,destination=/gocache" \
    --mount "type=volume,source=cache,destination=/home/.cache" \
    --mount "type=volume,source=crates,destination=/home/.cargo/registry" \
    --mount "type=volume,source=git-crates,destination=/home/.cargo/git" \
    ${CONDITIONAL_HOST_MOUNTS} \
    -w "${MOUNT_DEST}" "${IMG}" "$@"
