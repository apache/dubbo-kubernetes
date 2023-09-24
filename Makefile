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

# Git information
GIT_VERSION ?= $(shell git describe --tags --always)
GIT_COMMIT_HASH ?= $(shell git rev-parse HEAD)
GIT_TREESTATE = "clean"
GIT_DIFF = $(shell git diff --quiet >/dev/null 2>&1; if [ $$? -eq 1 ]; then echo "1"; fi)
ifeq ($(GIT_DIFF), 1)
    GIT_TREESTATE = "dirty"
endif

BUILDDATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

LDFLAGS = "-X github.com/apache/dubbo-kubernetes/pkg/version.gitTag=$(GIT_VERSION) \
                      -X github.com/apache/dubbo-kubernetes/pkg/version.gitCommit=$(GIT_COMMIT_HASH) \
                      -X github.com/apache/dubbo-kubernetes/pkg/version.gitTreeState=$(GIT_TREESTATE) \
                      -X github.com/apache/dubbo-kubernetes/pkg/version.buildDate=$(BUILDDATE)"

# Images management
REGISTRY ?= docker.io
REGISTRY_NAMESPACE ?= apache
REGISTRY_USER_NAME?=""
REGISTRY_PASSWORD?=""

# Image URL to use all building/pushing image targets
DUBBO_CP_IMG ?= "${REGISTRY}/${REGISTRY_NAMESPACE}/dubbo-cp:${GIT_VERSION}"
DUBBO_UI_IMG ?= "${REGISTRY}/${REGISTRY_NAMESPACE}/dubbo-ui:${GIT_VERSION}"
DUBBO_DUBBOCTL_BUILDX_DIR ?= "./bin/dubboctl"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
SWAGGER ?= $(LOCALBIN)/swag
GOLANG_LINT ?= $(LOCALBIN)/golangci-lint
GOFUMPT  ?= $(LOCALBIN)/gofumpt


## Tool Versions
SWAGGER_VERSION ?= v1.16.1
GOLANG_LINT_VERSION ?= v1.52.2
GOFUMPT_VERSION ?= latest
## docker buildx support platform
PLATFORMS ?= linux/arm64,linux/amd64


##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: swagger
swagger: swagger-install ## Generate dubbocp swagger docs.
	$(SWAGGER) init --parseDependency -d app/dubbo-cp,pkg/admin -o hack/swagger
	@rm -f hack/swagger/docs.go hack/swagger/swagger.yaml

.PHONY: fmt
fmt: gofumpt-install ## Run gofumpt against code.
	$(GOFUMPT) -l -w .

.PHONY: vet
vet: ## Run go vet against code.
	@find . -type f -name '*.go'| grep -v "/vendor/" | xargs gofmt -w -s

# Run mod tidy against code
.PHONY: tidy
tidy:
	@go mod tidy

.PHONY: lint
lint: golangci-lint-install  ## Run golang lint against code
	GO111MODULE=on $(GOLANG_LINT) run ./... --timeout=30m -v  --disable-all --enable=gofumpt --enable=govet --enable=staticcheck --enable=ineffassign --enable=misspell

.PHONY: test
test: fmt vet  ## Run all tests.
	go test -coverprofile coverage.out -covermode=atomic ./...


.PHONY: test-dubboctl
test-dubboctl: fmt vet  ## Run tests for dubboctl
	go test -coverprofile coverage.out -covermode=atomic github.com/apache/dubbo-kubernetes/app/dubboctl/...

.PHONY: test-dubbocp
test-dubbocp: fmt vet  ## Run tests for dubbo control-plane
	go test -coverprofile coverage.out -covermode=atomic github.com/apache/dubbo-kubernetes/pkg/...


.PHONY: echoLDFLAGS
echoLDFLAGS:
	@echo $(LDFLAGS)

.PHONY: build
build: build-dubbocp build-dubboctl ## Build binary with the dubbo control-plane and dubboctl

.PHONY: all
all: test build

.PHONY: build-dubbocp
build-dubbocp:  ## Build binary with the dubbo control plane.
	GOOS=$(GOOS) go build -ldflags $(LDFLAGS) -o bin/dubbo-cp app/dubbo-cp/main.go

.PHONY: build-dubboctl
build-dubboctl: ## Build binary with the dubbo dubboctl.
	CGO_ENABLED=0 GOOS=$(GOOS) go build -ldflags $(LDFLAGS) -o bin/dubboctl app/dubboctl/main.go


.PHONY: build-ui
build-ui: $(LOCALBIN)## Build the distribution of the dubbocp ui pages.
	docker build --build-arg LDFLAGS=$(LDFLAGS) --build-arg PKGNAME=ui -t ${DUBBO_UI_IMG} ./ui
	docker run -d --name dubbo-ui ${DUBBO_UI_IMG}
	docker cp dubbo-ui:/usr/share/nginx/html/ $(LOCALBIN)/ui
	rm -f -R ./app/dubbo-ui/dist/*
	rm -f ./bin/ui/50x.html
	mkdir -p ./app/dubbo-ui/dist
	cp -R ./bin/ui/* ./app/dubbo-ui/dist/
	rm -f -R ./bin/ui

.PHONY: image
image: image-dubbocp  image-ui ## Build docker image with the dubbocp dubbo-ui

.PHONY: image-dubbocp
image-dubbocp: ## Build docker image with the dubbocp.
	docker build --build-arg LDFLAGS=$(LDFLAGS) --build-arg PKGNAME=dubbo-cp -t ${DUBBO_CP_IMG} .


.PHONY: image-ui
image-ui: ## Build docker image with the dubbo ui.
	docker build --build-arg LDFLAGS=$(LDFLAGS) --build-arg PKGNAME=dubbo-ui -t ${DUBBO_UI_IMG} ./ui



.PHONY: buildx
buildx: buildx-dubbocp ## Build and push docker cross-platform image for the dubbo control-plane


.PHONY: buildx-dubbocp
buildx-dubbocp:  ## Build and push docker image with the dubbo control plane for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile_dubbocp.cross
	- docker buildx create --name project-dubbo-cp-builder
	docker buildx use project-dubbo-cp-builder
	- docker buildx build --build-arg LDFLAGS=$(LDFLAGS) --push --platform=$(PLATFORMS) --tag ${DUBBO_CP_IMG} -f Dockerfile_dubbocp.cross .
	- docker buildx rm project-dubbo-cp-builder
	rm Dockerfile_dubbocp.cross

.PHONY: buildx-dubboctl
buildx-dubboctl:  ## Build the dubboctl distribution for cross-platform support
	@rm -f -R $(DUBBO_DUBBOCTL_BUILDX_DIR)
	@mkdir $(DUBBO_DUBBOCTL_BUILDX_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags $(LDFLAGS) -o $(DUBBO_DUBBOCTL_BUILDX_DIR)/linux/amd64/dubboctl app/dubboctl/main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags $(LDFLAGS) -o $(DUBBO_DUBBOCTL_BUILDX_DIR)/linux/arm64/dubboctl app/dubboctl/main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags $(LDFLAGS) -o $(DUBBO_DUBBOCTL_BUILDX_DIR)/darwin/amd64/dubboctl app/dubboctl/main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags $(LDFLAGS) -o $(DUBBO_DUBBOCTL_BUILDX_DIR)/darwin/arm64/dubboctl app/dubboctl/main.go
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags $(LDFLAGS) -o $(DUBBO_DUBBOCTL_BUILDX_DIR)/windows/amd64/dubboctl.exe app/dubboctl/main.go

	tar -cvzf $(DUBBO_DUBBOCTL_BUILDX_DIR)/dubboctl-${GIT_VERSION}-linux-amd64.tar.gz  -C $(DUBBO_DUBBOCTL_BUILDX_DIR)/linux/amd64/ dubboctl
	tar -cvzf $(DUBBO_DUBBOCTL_BUILDX_DIR)/dubboctl-${GIT_VERSION}-linux-arm64.tar.gz  -C $(DUBBO_DUBBOCTL_BUILDX_DIR)/linux/arm64/ dubboctl

	tar -cvzf $(DUBBO_DUBBOCTL_BUILDX_DIR)/dubboctl-${GIT_VERSION}-osx-arm64.tar.gz  -C $(DUBBO_DUBBOCTL_BUILDX_DIR)/darwin/arm64/ dubboctl
	tar -cvzf $(DUBBO_DUBBOCTL_BUILDX_DIR)/dubboctl-${GIT_VERSION}-osx.tar.gz  -C $(DUBBO_DUBBOCTL_BUILDX_DIR)/darwin/amd64/ dubboctl
	zip  $(DUBBO_DUBBOCTL_BUILDX_DIR)/dubboctl-${GIT_VERSION}-win.zip -D -j $(DUBBO_DUBBOCTL_BUILDX_DIR)/windows/amd64/dubboctl.exe



.PHONY: push-images
push-images: push-image-dubbocp push-image-ui

.PHONY: push-image-dubbocp
push-image-dubbocp: ## Push dubbocp images.
ifneq ($(REGISTRY_USER_NAME), "")
	docker login -u $(REGISTRY_USER_NAME) -p $(REGISTRY_PASSWORD) ${REGISTRY}
endif
	docker push ${DUBBO_CP_IMG}


.PHONY: push-image-ui
push-image-ui: ## Push dubbocp ui images.
ifneq ($(REGISTRY_USER_NAME), "")
	docker login -u $(REGISTRY_USER_NAME) -p $(REGISTRY_PASSWORD) ${REGISTRY}
endif
	docker push ${DUBBO_UI_IMG}




.PHONY: swagger-install
swagger-install: $(LOCALBIN) ## Download swagger locally if necessary.
	test -s $(LOCALBIN)/swag  || \
	GOBIN=$(LOCALBIN) go install  github.com/swaggo/swag/cmd/swag@$(SWAGGER_VERSION)


GOLANG_LINT_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh"
.PHONY: golangci-lint-install
golangci-lint-install: $(LOCALBIN) ## Download golangci lint locally if necessary.
	test -s $(LOCALBIN)/golangci-lint  && $(LOCALBIN)/golangci-lint --version | grep -q $(GOLANG_LINT_VERSION) || \
	GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANG_LINT_VERSION)


.PHONY: gofumpt-install
gofumpt-install: $(LOCALBIN) ## Download gofumpt locally if necessary.
	test -s $(LOCALBIN)/gofumpt || \
	GOBIN=$(LOCALBIN) go install mvdan.cc/gofumpt@$(GOFUMPT_VERSION)