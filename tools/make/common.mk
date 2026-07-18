# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#	http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# ==============================================================================
# Shared configuration (override on the command line, e.g. `make build GOOS=linux`)
# ==============================================================================

GOOS        ?= $(shell go env GOOS)
GOARCH      ?= $(shell go env GOARCH)
GIT_VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo unknown)
BIN_DIR     ?= bin

GOTESTFLAGS           ?= -race
GOLANGCI_LINT_VERSION ?= v2.6.2

HUB       ?= kdubbo
IMAGE_TAG ?= debug
IMAGE     ?= $(HUB)/dubbod:$(IMAGE_TAG)

VERSION_PKG := github.com/apache/dubbo-kubernetes/pkg/version
LDFLAGS     ?= -X $(VERSION_PKG).gitTag=$(GIT_VERSION)
GO_BUILD     = CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags "$(LDFLAGS)"

CHARTS := manifests/charts/base manifests/charts/dubbod

.DEFAULT_GOAL := build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} \
		/^[a-zA-Z_0-9%-]+:.*?##/ { printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)

.PHONY: verify
verify: check-fmt lint test lint-helm ## Everything fast enough to run before pushing.
