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

# Build the image binary
FROM golang:1.20.1-alpine3.17 as builder


# Build argments
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG LDFLAGS="-s -w"
ARG BUILD

WORKDIR /go/src/github.com/apache/dubbo-kubernetes

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

#RUN if [[ "${PKGNAME}" == "authority" ]]; then apk --update add gcc libc-dev upx ca-certificates && update-ca-certificates; fi

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN if [[ "${BUILD}" != "CI" ]]; then go env -w GOPROXY=https://goproxy.cn,direct; fi
RUN go env
RUN go mod download


# Copy the go source
COPY pkg pkg/
COPY app app/
COPY api api/
COPY conf conf/
COPY deploy deploy/
COPY generate generate/
COPY test/proxy/dp test/proxy/dp

# Build
RUN env
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="${LDFLAGS:- -s -w}" -a -o dubboctl /go/src/github.com/apache/dubbo-kubernetes/app/dubboctl/main.go

FROM envoyproxy/envoy:v1.29.2



# Build
WORKDIR /
ARG PKGNAME
COPY --from=builder /go/src/github.com/apache/dubbo-kubernetes/dubboctl .
COPY --from=builder /go/src/github.com/apache/dubbo-kubernetes/test/proxy/dp .
#COPY --from=builder /go/src/github.com/apache/dubbo-kubernetes/conf/admin.yml .
#ENV ADMIN_CONFIG_PATH=./admin.yml


ENTRYPOINT ["./dubboctl", "proxy","--proxy-type=ingress","--dataplane-file=/ingress.yaml"]
