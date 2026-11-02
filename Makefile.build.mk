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

BINARIES            := dubboctl dubbod
main_path_dubboctl  := dubboctl/main.go
main_path_dubbod    := dubbod/discovery/cmd/main.go
DUBBOD_BASE_IMAGE   ?=
DOCKER_DIR          = $(BIN_DIR)/docker/dubbod

##@ Build

.PHONY: build
build: $(addprefix build-,$(BINARIES)) ## Build all binaries into bin/ (host OS/arch; override with GOOS=/GOARCH=). Single binary: make build-<name>.

.PHONY: $(addprefix build-,$(BINARIES))
$(addprefix build-,$(BINARIES)): build-%:
	$(GO_BUILD) -o $(BIN_DIR)/$* $(main_path_$*)

.PHONY: docker-base
docker-base: ## Build base images declared in tools/docker.yaml (HUB=/TAG=/PUSH=).
	HUB=$(HUB) TAG=$(TAG) ./tools/build-base-images.sh

.PHONY: docker-dubbod
docker-dubbod: ## Build a local Linux dubbod image (REPO=/TAG=; BUILD_ALL=false builds only dubbod).
	$(MAKE) $(addprefix build-,$(if $(filter true,$(BUILD_ALL)),$(BINARIES),dubbod)) GOOS=linux BIN_DIR=$(DOCKER_DIR)/$(GOARCH)
	docker build --platform linux/$(GOARCH) $(if $(DUBBOD_BASE_IMAGE),--build-arg BASE_IMAGE=$(DUBBOD_BASE_IMAGE)) -f dubbod/discovery/docker/dockerfile.dubbod -t $(IMAGE) $(DOCKER_DIR)

.PHONY: clone-examples
clone-examples: ## Copy examples/ next to the binaries for release packaging.
	mkdir -p $(BIN_DIR)
	cp -r examples $(BIN_DIR)/examples

.PHONY: clean
clean: ## Remove build artifacts.
	rm -rf $(BIN_DIR)