# hadolint global ignore=SC2086

ARG GOLANG_IMAGE=golang:1.24-bookworm
# hadolint ignore=DL3006
FROM ${GOLANG_IMAGE} AS binary_tools_context_base
# TARGETARCH is an automatic platform ARG enabled by Docker BuildKit.
ARG TARGETARCH

# Dubbo Kubernetes version/SHA for kubetype-gen
ARG DUBBO_KUBERNETES_VERSION=main

# Pinned versions for code generators
ENV K8S_CODE_GENERATOR_VERSION=1.29.4

ENV GO111MODULE=on
ENV GOPROXY=https://proxy.golang.org

WORKDIR /tmp
ENV GOPATH=/tmp/go
# Avoid any attempts to automatically download a new Go version
ENV GOTOOLCHAIN=local

ENV OUTDIR=/out
RUN mkdir -p ${OUTDIR}/usr/bin
RUN mkdir -p ${OUTDIR}/usr/local

# Update distro and install dependencies
# hadolint ignore=DL3042
RUN --mount=type=cache,target=/var/lib/apt/lists,sharing=locked \
    --mount=type=cache,target=/var/cache/apt,sharing=locked \
    rm -f /etc/apt/apt.conf.d/docker-clean \
    && apt-get update \
    && apt-get -y --no-install-recommends install \
    build-essential \
    ca-certificates \
    curl \
    git \
    unzip \
    xz-utils

# Cleanup stuff we don't need in the final image
RUN rm -fr /usr/local/go/doc
RUN rm -fr /usr/local/go/test
RUN rm -fr /usr/local/go/api
RUN rm -fr /usr/local/go/bin/godoc
RUN rm -fr /usr/local/go/bin/gofmt

FROM binary_tools_context_base AS go_tools_1

# Build and install Kubernetes code generators
RUN --mount=type=cache,target=/tmp/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go install -ldflags="-extldflags -static -s -w" \
    k8s.io/code-generator/cmd/applyconfiguration-gen@kubernetes-${K8S_CODE_GENERATOR_VERSION} \
    k8s.io/code-generator/cmd/defaulter-gen@kubernetes-${K8S_CODE_GENERATOR_VERSION} \
    k8s.io/code-generator/cmd/client-gen@kubernetes-${K8S_CODE_GENERATOR_VERSION} \
    k8s.io/code-generator/cmd/lister-gen@kubernetes-${K8S_CODE_GENERATOR_VERSION} \
    k8s.io/code-generator/cmd/informer-gen@kubernetes-${K8S_CODE_GENERATOR_VERSION} \
    k8s.io/code-generator/cmd/deepcopy-gen@kubernetes-${K8S_CODE_GENERATOR_VERSION} \
    k8s.io/code-generator/cmd/go-to-protobuf@kubernetes-${K8S_CODE_GENERATOR_VERSION}

FROM binary_tools_context_base AS go_tools_2

# Build Dubbo Kubernetes kubetype-gen tool from source
# Copy pre-built kubetype-gen binary from project root
# IMPORTANT: Build context must be project root (where kubetype-gen exists)
# IMPORTANT: kubetype-gen must be built for Linux architecture (not macOS)
# Build command: cd tools/cmd/kubetype-gen && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-extldflags -static -s -w" -o ../../../kubetype-gen .
# For ARM64: use GOARCH=arm64 instead of amd64
# Run: cd /path/to/dubbo-kubernetes && nerdctl build -f tools/build-tools/docker/Dockerfile -t build-tools .
COPY --chmod=755 kubetype-gen /tmp/go/bin/kubetype-gen

FROM ubuntu:noble AS base_os_context

ENV DEBIAN_FRONTEND=noninteractive
ENV OUTDIR=/out

# Install minimal dependencies for code generation
# hadolint ignore=DL3008
RUN --mount=type=cache,target=/var/lib/apt/lists,sharing=locked \
    --mount=type=cache,target=/var/cache/apt,sharing=locked \
    rm -f /etc/apt/apt.conf.d/docker-clean \
    && apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    git \
    make

# Clean up stuff we don't need in the final image
RUN rm -rf /var/lib/apt/lists/* \
  && rm -fr /usr/share/doc \
  && rm -fr /usr/share/man \
  && rm -fr /tmp/*

FROM scratch AS build_tools

# Version from build arguments
ARG VERSION

# Labels used by Docker
LABEL "org.apache.dubbo.repo"="https://github.com/apache/dubbo-kubernetes"
LABEL "org.apache.dubbo.version"="${VERSION}"

# General
ENV HOME=/home
ENV LANG=C.UTF-8

# Go support
ENV GO111MODULE=on
# Avoid any attempts to automatically download a new Go version
ENV GOTOOLCHAIN=local
ENV GOPROXY=https://proxy.golang.org
ENV GOSUMDB=sum.golang.org
ENV GOROOT=/usr/local/go
ENV GOPATH=/go
ENV GOCACHE=/gocache
ENV GOBIN=/gobin
ENV PATH=/usr/local/go/bin:/gobin:/usr/bin:$PATH

# Create the file system
COPY --link --from=base_os_context / /
COPY --link --from=binary_tools_context_base /out/ /
COPY --link --from=binary_tools_context_base /usr/local/go /usr/local/go
COPY --link --from=go_tools_1 /tmp/go/bin/* /usr/bin/
COPY --link --from=go_tools_2 /tmp/go/bin/* /usr/bin/

# Create mountpoints
RUN mkdir -p /go && \
    mkdir -p /gocache && \
    mkdir -p /gobin && \
    mkdir -p /home/.cache

RUN chmod -R 777 /go && \
    chmod -R 777 /gocache && \
    chmod -R 777 /gobin && \
    chmod -R 777 /home/.cache

WORKDIR /
