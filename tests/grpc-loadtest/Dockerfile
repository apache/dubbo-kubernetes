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

FROM golang:1.24-alpine AS builder

WORKDIR /build

# Install protoc and dependencies
RUN apk add --no-cache protobuf protoc protobuf-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Generate proto code
WORKDIR /build/proto
RUN chmod +x gen.sh && ./gen.sh

# Build server
WORKDIR /build
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./server

# Build client
RUN CGO_ENABLED=0 GOOS=linux go build -o client ./client

# Final image
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /build/server .
COPY --from=builder /build/client .

ENTRYPOINT []

