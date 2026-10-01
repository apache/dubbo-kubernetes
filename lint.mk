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

##@ Lint

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
