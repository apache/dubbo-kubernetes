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

.PHONY: proto build server client docker clean test

# Generate proto code
proto:
	cd proto && chmod +x gen.sh && ./gen.sh

# Download dependencies
deps:
	go mod download
	go mod tidy

# Build server
server: proto deps
	go build -o server ./server

# Build client
client: proto deps
	go build -o client ./client

# Build both
build: server client

# Build Docker images
docker:
	docker build -t loadtest-server:latest -f Dockerfile .
	docker build -t loadtest-client:latest -f Dockerfile .

# Clean build artifacts
clean:
	rm -f server client
	rm -f proto/*.pb.go
	rm -f proto/*_grpc.pb.go

# Run tests locally (requires servers to be running)
test-baseline:
	./client -target=localhost:8080 -mode=baseline -qps=10 -duration=10s

test-envoy:
	./client -target=localhost:8080 -mode=envoy -qps=10 -duration=10s

test-xds:
	./client -target=xds:///localhost:8080 -mode=xds -qps=10 -duration=10s

# All tests
test: test-baseline test-envoy test-xds

