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

.PHONY: build-dubboctl
build-dubboctl:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
	-ldflags "-X github.com/apache/dubbo-kubernetes/pkg/version.gitTag=$(GIT_VERSION)" \
    -o bin/dubboctl dubboctl/main.go

.PHONY: clone-sample
clone-sample:
	mkdir -p bin
	cp -r samples bin/samples

.PHONY: build-planet
build-planet:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -buildvcs=false -o bin/planet-discovery ./dubbod/planet/cmd/planet-discovery

	nerdctl build --build-arg GOOS=linux --build-arg GOARCH=arm64 \
		-f dubbod/planet/docker/dockerfile.planet \
		-t mfordjody/planet:0.3.0-debug \
		.

	nerdctl push mfordjody/planet:0.3.0-debug