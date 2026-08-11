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

##@ Lint / Format

.PHONY: lint
lint: lint-go lint-shell ## Run Go and shell linters.

.PHONY: lint-go
lint-go: ## Run golangci-lint.
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint not found; install $(GOLANGCI_LINT_VERSION) from https://golangci-lint.run/welcome/install/"; \
		exit 1; \
	fi
	golangci-lint run ./...

.PHONY: lint-shell
lint-shell: ## Run shellcheck over all shell scripts (skipped if not installed).
	@if command -v shellcheck >/dev/null 2>&1; then \
		find . -name '*.sh' -not -path './vendor/*' -not -path './$(BIN_DIR)/*' -print0 | xargs -0 -r shellcheck; \
	else \
		echo "shellcheck not found, skipping shell lint"; \
	fi

.PHONY: lint-helm
lint-helm: ## Lint and render the Helm charts.
	@for chart in $(CHARTS); do \
		helm lint $$chart || exit 1; \
		helm template test-release $$chart >/dev/null || exit 1; \
	done
	@if grep -q 'authorizationpolicys.security.dubbo.apache.org' manifests/charts/base/files/crd-all.gen.yaml; then \
		echo "AuthorizationPolicy CRD uses invalid plural authorizationpolicys"; \
		exit 1; \
	fi
	@grep -q 'name: authorizationpolicies.security.dubbo.apache.org' manifests/charts/base/files/crd-all.gen.yaml || { \
		echo "AuthorizationPolicy CRD plural must be authorizationpolicies"; \
		exit 1; \
	}

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
