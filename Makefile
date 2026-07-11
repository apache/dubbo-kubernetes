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

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
GIT_VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo unknown)

GOLANGCI_LINT_VERSION ?= v2.6.2
GOTESTFLAGS ?= -race

.PHONY: default
default: build

# ------------------------------------------------------------------------
# Build
# ------------------------------------------------------------------------

.PHONY: build
build: build-dubboctl build-dubbod build-dubbo-cni

.PHONY: build-dubboctl
build-dubboctl:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
	-ldflags "-X github.com/apache/dubbo-kubernetes/pkg/version.gitTag=$(GIT_VERSION)" \
    -o bin/dubboctl cli/main.go

.PHONY: build-dubbod
build-dubbod:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
    -o bin/dubbod dubbod/discovery/cmd/main.go

.PHONY: build-dubbo-cni
build-dubbo-cni:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
    -o bin/dubbo-cni cni/main.go

.PHONY: clone-sample
clone-sample:
	mkdir -p bin
	cp -r samples bin/samples

# ------------------------------------------------------------------------
# Test
# ------------------------------------------------------------------------

.PHONY: test
test:
	go test $(GOTESTFLAGS) ./...

.PHONY: test-coverage
test-coverage:
	go test $(GOTESTFLAGS) -coverprofile=coverage.txt -covermode=atomic ./...

# ------------------------------------------------------------------------
# Lint / format
# ------------------------------------------------------------------------

.PHONY: lint
lint: lint-go lint-shell

.PHONY: lint-go
lint-go:
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint not found; install $(GOLANGCI_LINT_VERSION) from https://golangci-lint.run/welcome/install/"; \
		exit 1; \
	fi
	golangci-lint run ./...

.PHONY: lint-shell
lint-shell:
	@if command -v shellcheck >/dev/null 2>&1; then \
		find . -name '*.sh' -not -path './vendor/*' -not -path './bin/*' -print0 | xargs -0 -r shellcheck; \
	else \
		echo "shellcheck not found, skipping shell lint"; \
	fi

.PHONY: fmt
fmt:
	gofmt -s -w .

.PHONY: check-fmt
check-fmt:
	@out="$$(gofmt -s -l .)"; \
	if [ -n "$$out" ]; then \
		echo "The following files need 'gofmt -s':"; \
		echo "$$out"; \
		exit 1; \
	fi

# ------------------------------------------------------------------------
# Hygiene gates (CI runs these; they must leave the tree clean)
# ------------------------------------------------------------------------

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: check-clean-repo
check-clean-repo:
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "The working tree is dirty after running generators/tidy:"; \
		git status --porcelain; \
		git diff; \
		exit 1; \
	fi

.PHONY: check-tidy
check-tidy: tidy check-clean-repo

# ------------------------------------------------------------------------
# Helm
# ------------------------------------------------------------------------

CHARTS := manifests/charts/base manifests/charts/dubbod

.PHONY: lint-helm
lint-helm:
	@for chart in $(CHARTS); do \
		helm lint $$chart || exit 1; \
		helm template test-release $$chart >/dev/null || exit 1; \
	done

# ------------------------------------------------------------------------
# Misc
# ------------------------------------------------------------------------

.PHONY: clean
clean:
	rm -rf bin coverage.txt
