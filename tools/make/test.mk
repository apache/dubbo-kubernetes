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

.PHONY: test
test: ## Run unit tests (GOTESTFLAGS defaults to -race).
	go test $(GOTESTFLAGS) ./...

.PHONY: test-coverage
test-coverage: ## Run unit tests and write coverage.txt.
	go test $(GOTESTFLAGS) -coverprofile=coverage.txt -covermode=atomic ./...

.PHONY: test-e2e
test-e2e: ## Run the kind smoke test (needs docker, kind, kubectl, helm; knobs in tests/e2e/run.sh).
	bash tests/e2e/run.sh
