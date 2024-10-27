GO = go
GO_INSTALL = $(GO) install
GO_BIN = $(shell go env GOPATH)/bin

.PHONY: fmt
fmt: gofmt dubbogofmt golangci-lint-fmt fmt/proto ## Dev: Run various format tools

.PHONY: gofmt
	go fmt ./...

.PHONY: fmt/proto
fmt/proto: ## Dev: Run clang-format on .proto files
	find . -name '*.proto' | xargs -L 1 $(CLANG_FORMAT) -i

.PHONY: tidy
tidy:
	@TOP=$(shell pwd) && \
	for m in $$(find . -name go.mod) ; do \
		( cd $$(dirname $$m) && go mod tidy ) ; \
	done

.PHONY: shellcheck
shellcheck:
	find . -name "*.sh" -not -path "./.git/*" -exec $(SHELLCHECK) -P SCRIPTDIR -x {} +

.PHONY: golangci-lint
golangci-lint: ## Dev: Runs golangci-lint linter
ifndef CI
	GOMEMLIMIT=7GiB $(GOENV) $(GOLANGCI_LINT) run --timeout=10m -v
else
	@echo "skipping golangci-lint as it's done as a github action"
endif

.PHONY: golangci-lint-fmt
golangci-lint-fmt:
	GOMEMLIMIT=7GiB $(GOENV) $(GOLANGCI_LINT) run --timeout=10m -v \
		--disable-all \
		--enable gofumpt

.PHONY: dubbogofmt
dubbogofmt: $(GO_BIN)/imports-formatter
	GOROOT=$(shell go env GOROOT) $(GO_BIN)/imports-formatter

$(GO_BIN)/imports-formatter:
	$(GO_INSTALL) github.com/dubbogo/tools/cmd/imports-formatter@latest

.PHONY: helm-lint
helm-lint:
	find ./deploy/charts -maxdepth 1 -mindepth 1 -type d -exec $(HELM) lint --strict {} \;

.PHONY: ginkgo/unfocus
ginkgo/unfocus:
	@$(GINKGO) unfocus

.PHONY: ginkgo/lint
ginkgo/lint:
	go run $(TOOLS_DIR)/ci/check_test_files.go

.PHONY: format/common
format/common: generate tidy ginkgo/unfocus

.PHONY: format
format: fmt format/common

.PHONY: hadolint
hadolint:
	find ./tools/releases/dockerfiles/ -type f -iname "Dockerfile*" | grep -v dockerignore | xargs -I {} $(HADOLINT) {}

.PHONY: lint
lint: helm-lint golangci-lint shellcheck hadolint ginkgo/lint

.PHONY: check
check: format/common lint ## Dev: Run code checks (go fmt, go vet, ...)
	@untracked() { git ls-files --other --directory --exclude-standard --no-empty-directory; }; \
	check-changes() { git --no-pager diff "$$@"; }; \
	if [ $$(untracked | wc -l) -gt 0 ]; then \
		FAILED=true; \
		echo "The following files are untracked:"; \
		untracked; \
	fi; \
	if [ $$(check-changes --name-only | wc -l) -gt 0 ]; then \
		FAILED=true; \
		echo "The following changes (result of code generators and code checks) have been detected:"; \
		check-changes; \
	fi; \
	if [ "$$FAILED" = true ]; then exit 1; fi
