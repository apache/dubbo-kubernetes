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

# Binary name -> main package mapping. Adding a binary is one line here plus
# appending it to BINARIES.
BINARIES            := dubboctl dubbod
main_path_dubboctl  := dubboctl/main.go
main_path_dubbod    := dubbod/discovery/cmd/main.go
DUBBOD_BASE_IMAGE   ?=
# Directory containing precompiled linux-amd64/triple-engine and/or linux-arm64/triple-engine.
TRIPLE_ENGINE_BINARY_DIR ?=

##@ Build

.PHONY: build
build: $(addprefix build-,$(BINARIES)) ## Build all binaries into bin/ (host OS/arch; override with GOOS=/GOARCH=). Single binary: make build-<name>.

.PHONY: $(addprefix build-,$(BINARIES))
$(addprefix build-,$(filter-out dubbod,$(BINARIES))): build-%:
	$(GO_BUILD) -o $(BIN_DIR)/$* $(main_path_$*)

build-dubbod:
	@if [ -n "$(TRIPLE_ENGINE_BINARY_DIR)" ]; then \
		if [ ! -f "$(TRIPLE_ENGINE_BINARY_DIR)/linux-$(GOARCH)/triple-engine" ]; then \
			echo "Missing precompiled Triple binary: $(TRIPLE_ENGINE_BINARY_DIR)/linux-$(GOARCH)/triple-engine" >&2; exit 1; \
		fi; \
		lock="$(CURDIR)/dubbod/discovery/pkg/bootstrap/triple-binaries/build-lock"; \
		mkdir "$$lock" || { echo "Another dubbod embedding build owns $$lock" >&2; exit 1; }; \
		stage=; \
		trap 'if [ -n "$$stage" ]; then rm -rf "$$stage"; fi; rmdir "$$lock"' EXIT; \
		trap 'exit 130' INT; trap 'exit 143' TERM; \
		stage=$$(mktemp -d "$(CURDIR)/dubbod/discovery/pkg/bootstrap/triple-binaries/build-XXXXXX") || exit 1; \
		for arch in amd64 arm64; do \
			if [ -f "$(TRIPLE_ENGINE_BINARY_DIR)/linux-$$arch/triple-engine" ]; then \
				cp "$(TRIPLE_ENGINE_BINARY_DIR)/linux-$$arch/triple-engine" "$$stage/linux-$$arch.bin" || exit 1; \
			fi; \
		done; \
		$(GO_BUILD) -tags=triple_engine -o $(BIN_DIR)/dubbod $(main_path_dubbod); \
	else \
		$(GO_BUILD) -o $(BIN_DIR)/dubbod $(main_path_dubbod); \
	fi

.PHONY: docker-build docker.dubbod
docker.dubbod: ## Build a local Linux dubbod image (REPO=/TAG=; BUILD_ALL=false builds only dubbod).
	@test -n "$(TRIPLE_ENGINE_BINARY_DIR)" || { echo "TRIPLE_ENGINE_BINARY_DIR must point to the precompiled Linux Triple binaries" >&2; exit 1; }
	$(MAKE) $(addprefix build-,$(if $(filter true,$(BUILD_ALL)),$(BINARIES),dubbod)) GOOS=linux BIN_DIR=$(BIN_DIR)/docker/dubbod/$(GOARCH)
	docker build --platform linux/$(GOARCH) $(if $(DUBBOD_BASE_IMAGE),--build-arg BASE_IMAGE=$(DUBBOD_BASE_IMAGE)) -f dubbod/discovery/docker/dockerfile.dubbod -t $(IMAGE) $(BIN_DIR)/docker/dubbod

docker-build: docker.dubbod ## Build the dubbod container image (override with HUB=/TAG= or IMAGE_TAG=).

.PHONY: clone-examples
clone-examples: ## Copy examples/ next to the binaries for release packaging.
	mkdir -p $(BIN_DIR)
	cp -r examples $(BIN_DIR)/examples

.PHONY: clean
clean: ## Remove build artifacts.
	rm -rf $(BIN_DIR)
