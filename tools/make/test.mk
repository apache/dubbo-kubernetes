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

##@ Test

BENCHMARK_PACKAGES ?= ./pkg/kube/krt ./dubbod/discovery/pkg/xds
BENCHMARK_PATTERN  ?= Benchmark(KRTFetch|ClientsForPushProxylessTargeted|Proxyless(Push|FullPush)Scale)
BENCHMARK_TIME     ?= 1s
BENCHMARK_COUNT    ?= 3
BENCHMARK_CPU      ?= 1

.PHONY: test
test: ## Run unit tests (GOTESTFLAGS defaults to -race).
	go test $(GOTESTFLAGS) ./...

.PHONY: test-coverage
test-coverage: ## Run unit tests and write coverage.txt.
	go test $(GOTESTFLAGS) -coverprofile=coverage.txt -covermode=atomic ./...

.PHONY: test-e2e
test-e2e: ## Run the kind smoke test (needs docker, kind, kubectl, helm; knobs in tests/e2e/run.sh).
	bash tests/e2e/run.sh

.PHONY: benchmark
benchmark: ## Run repeatable control-plane microbenchmarks (override BENCHMARK_* variables).
	go test $(BENCHMARK_PACKAGES) -run '^$$' -bench '$(BENCHMARK_PATTERN)' -benchmem -benchtime=$(BENCHMARK_TIME) -count=$(BENCHMARK_COUNT) -cpu=$(BENCHMARK_CPU)

.PHONY: benchmark-smoke
benchmark-smoke: ## Run representative targeted/full xDS scale cases as a correctness gate.
	go test ./dubbod/discovery/pkg/xds -run '^TestProxylessPushBenchmarkFixture$$' -bench '^BenchmarkProxyless(Push|FullPush)Scale$$/^services=100$$/^endpoints_per_service=10$$/^connections=100$$' -benchmem -benchtime=1x -count=1 -cpu=1
