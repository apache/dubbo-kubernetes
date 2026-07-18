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
BINARIES            := dubboctl dubbod dubbo-cni
main_path_dubboctl  := dubboctl/main.go
main_path_dubbod    := dubbod/discovery/cmd/main.go
main_path_dubbo-cni := cni/main.go

##@ Build

.PHONY: build
build: $(addprefix build-,$(BINARIES)) ## Build all binaries into bin/ (host OS/arch; override with GOOS=/GOARCH=). Single binary: make build-<name>.

.PHONY: $(addprefix build-,$(BINARIES))
$(addprefix build-,$(BINARIES)): build-%:
	$(GO_BUILD) -o $(BIN_DIR)/$* $(main_path_$*)

.PHONY: docker-build
docker-build: ## Build the dubbod container image (override with HUB=/IMAGE_TAG=).
	docker build -f dubbod/discovery/docker/dockerfile.dubbod -t $(IMAGE) .

.PHONY: clone-sample
clone-sample: ## Copy samples/ next to the binaries for release packaging.
	mkdir -p $(BIN_DIR)
	cp -r samples $(BIN_DIR)/samples

.PHONY: clean
clean: ## Remove build artifacts and coverage output.
	rm -rf $(BIN_DIR) coverage.txt
