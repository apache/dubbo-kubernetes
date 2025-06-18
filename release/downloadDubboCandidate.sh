#!/bin/sh
set -e

# Determines the operating system.
OS="${TARGET_OS:-$(uname)}"
if [ "${OS}" = "Darwin" ] ; then
  OSEXT="osx"
else
  OSEXT="linux"
fi

# Package type, default to dubbo-cp
PACKAGE_TYPE="${PACKAGE_TYPE:-dubbo-cp}"

# Determine the latest Dubbo version by version number ignoring alpha, beta, and rc versions.
if [ "${DUBBO_VERSION}" = "" ] ; then
  DUBBO_VERSION="$(curl -s https://api.github.com/repos/apache/dubbo-kubernetes/releases | \
                       grep '"tag_name":' | \
                       grep -vE '(alpha|beta|rc)' | \
                       head -1 | \
                       sed -E 's/.*"([^"]+)".*/\1/')"
  DUBBO_VERSION="${DUBBO_VERSION##*/}"
fi

if [ "${DUBBO_VERSION}" = "" ] ; then
  printf "Unable to get latest Dubbo version. Set DUBBO_VERSION env var and re-run. For example: export DUBBO_VERSION=0.1.0\n"
  exit 1;
fi

LOCAL_ARCH=$(uname -m)
if [ "${TARGET_ARCH}" ]; then
    LOCAL_ARCH=${TARGET_ARCH}
fi

case "${LOCAL_ARCH}" in
  x86_64|amd64)
    DUBBO_ARCH=amd64
    ;;
  armv8*|aarch64*|arm64)
    DUBBO_ARCH=arm64
    ;;
  armv*)
    DUBBO_ARCH=armv7
    ;;
  *)
    echo "This system's architecture, ${LOCAL_ARCH}, isn't supported"
    exit 1
    ;;
esac

NAME="${PACKAGE_TYPE}-${DUBBO_VERSION}"
URL="https://github.com/apache/dubbo-kubernetes/releases/download/${DUBBO_VERSION}/${PACKAGE_TYPE}-${DUBBO_VERSION}-${OSEXT}.tar.gz"
ARCH_URL="https://github.com/apache/dubbo-kubernetes/releases/download/${DUBBO_VERSION}/${PACKAGE_TYPE}-${DUBBO_VERSION}-${OSEXT}-${DUBBO_ARCH}.tar.gz"

with_arch() {
  printf "\nDownloading %s from %s ...\n" "${NAME}" "$ARCH_URL"
  if ! curl -o /dev/null -sIf "$ARCH_URL"; then
    printf "\n%s is not found, please specify a valid DUBBO_VERSION and TARGET_ARCH\n" "$ARCH_URL"
    return 1
  fi
  filename="${PACKAGE_TYPE}-${DUBBO_VERSION}-${OSEXT}-${DUBBO_ARCH}.tar.gz"
  curl -fLO "$ARCH_URL"
  tar -xzf "${filename}"
  rm "${filename}"
  return 0
}

without_arch() {
  printf "\nDownloading %s from %s ...\n" "$NAME" "$URL"
  if ! curl -o /dev/null -sIf "$URL"; then
    printf "\n%s is not found, please specify a valid DUBBO_VERSION\n" "$URL"
    return 1
  fi
  filename="${PACKAGE_TYPE}-${DUBBO_VERSION}-${OSEXT}.tar.gz"
  curl -fLO "$URL"
  tar -xzf "${filename}"
  rm "${filename}"
  return 0
}

if ! with_arch; then
  if ! without_arch; then
    echo "Download failed."
    exit 1
  fi
fi

printf "%s download complete!\n" "${filename}"

# Print message.
printf "\n"
printf "Add the dubbo to your path with:\n"
printf "  export PATH=\$HOME/.dubbo/bin:\$PATH \n"
printf "\n"