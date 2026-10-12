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

##@ Build

.PHONY: build
build: $(addprefix build-,$(BINARIES)) ## Build all binaries into bin/ (host OS/arch; override with GOOS=/GOARCH=). Single binary: make build-<name>.

.PHONY: $(addprefix build-,$(BINARIES))
$(addprefix build-,$(BINARIES)): build-%:
	$(GO_BUILD) -o $(BIN_DIR)/$* $(main_path_$*)

.PHONY: docker-build docker.dubbod
docker.dubbod: ## Build a local Linux dubbod image (REPO=/TAG=; BUILD_ALL=false builds only dubbod).
	$(MAKE) $(addprefix build-,$(if $(filter true,$(BUILD_ALL)),$(BINARIES),dubbod)) GOOS=linux BIN_DIR=$(BIN_DIR)/docker/dubbod
	rm -rf $(BIN_DIR)/docker/dubbod/legal
	go run ./tools/licenses -binary dubbod -goos linux -goarch $(GOARCH) -out $(BIN_DIR)/docker/dubbod/legal
	docker build --platform linux/$(GOARCH) --build-arg BUILD_MODE=prebuilt -f dubbod/discovery/docker/dockerfile.dubbod -t $(IMAGE) $(BIN_DIR)/docker/dubbod

docker-build: ## Build the dubbod container image (override with HUB=/TAG= or IMAGE_TAG=).
	docker build -f dubbod/discovery/docker/dockerfile.dubbod -t $(IMAGE) .

.PHONY: clone-examples
clone-examples: ## Copy examples/ next to the binaries for release packaging.
	mkdir -p $(BIN_DIR)
	cp -r examples $(BIN_DIR)/examples

.PHONY: clean
clean: ## Remove build artifacts.
	rm -rf $(BIN_DIR)
