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

GOLANGCI_LINT_VERSION ?= v2.9.0

# Code generation. protoc-gen-go must match the google.golang.org/protobuf in
# go.mod, and protoc must match the version already recorded in the generated
# headers: a mismatch rewrites every file and turns `make check-generate` into
# a permanent failure.
PROTOC_VERSION         ?= v33.0
PROTOC_GEN_GO_VERSION  ?= v1.36.11
GOIMPORTS_VERSION      ?= latest
TOOL_BIN               ?= $(CURDIR)/$(BIN_DIR)/tools
OPERATOR_APIS_DIR      := operator/pkg/apis

HUB       ?= dubml
IMAGE_TAG ?= debug
IMAGE     ?= $(HUB)/dubbod:$(IMAGE_TAG)

VERSION_PKG := github.com/apache/dubbo-kubernetes/pkg/version
LDFLAGS     ?= -X $(VERSION_PKG).gitTag=$(GIT_VERSION) -X $(VERSION_PKG).buildVersion=$(GIT_VERSION)
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
verify: check-fmt lint lint-helm ## Everything fast enough to run before pushing.

##@ Format

.PHONY: fmt
fmt: ## Format all Go sources with gofmt -s.
	gofmt -s -w .

.PHONY: check-fmt
check-fmt: ## Fail if any Go source needs gofmt -s.
	@out="$$(gofmt -s -l .)"; \
	if [ -n "$$out" ]; then \
		echo "The following files need 'gofmt -s':"; \
		echo "$$out"; \
		exit 1; \
	fi

##@ Hygiene gates (CI runs these; they must leave the tree clean)

.PHONY: tidy
tidy: ## Run go mod tidy.
	go mod tidy

.PHONY: check-clean-repo
check-clean-repo: ## Fail if the working tree is dirty.
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "The working tree is dirty after running generators/tidy:"; \
		git status --porcelain; \
		git diff; \
		exit 1; \
	fi

.PHONY: check-tidy
check-tidy: tidy check-clean-repo ## go mod tidy, then fail if it changed anything.

.PHONY: generate
generate: generate-proto generate-schema ## Re-run every code generator.

.PHONY: generate-proto
generate-proto: ## Regenerate the operator values API from its .proto.
	@command -v protoc >/dev/null 2>&1 || { \
		echo "protoc not found; install $(PROTOC_VERSION) from https://github.com/protocolbuffers/protobuf/releases"; \
		exit 1; \
	}
	@GOBIN=$(TOOL_BIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)
	@cd $(OPERATOR_APIS_DIR) && PATH="$(TOOL_BIN):$$PATH" protoc \
		--proto_path=proto \
		--proto_path=$$(go list -f '{{ .Dir }}' -m k8s.io/api) \
		--proto_path=$$(go list -f '{{ .Dir }}' -m k8s.io/apimachinery) \
		--go_out=. proto/values_types.proto
	@# go_package is a full import path and protoc has no paths=source_relative
	@# here, so the output lands under a mirrored directory tree.
	mv $(OPERATOR_APIS_DIR)/dubbo.apache.org/dubbo/operator/pkg/apis/values_types.pb.go \
		$(OPERATOR_APIS_DIR)/values_types.pb.go
	rm -rf $(OPERATOR_APIS_DIR)/dubbo.apache.org

.PHONY: generate-schema
generate-schema: ## Regenerate the resource schema from metadata.yaml.
	@GOBIN=$(TOOL_BIN) go install golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION)
	@PATH="$(TOOL_BIN):$$PATH" go run ./pkg/config/schema/codegen/tools/collections.main.go

.PHONY: check-generate
check-generate: generate check-clean-repo ## Regenerate, then fail if anything changed.
	@echo "Generated sources are up to date."
