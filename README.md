# Dubbo Service Mesh for Kubernetes

[![Go Reference](https://pkg.go.dev/badge/github.com/apache/dubbo-kubernetes.svg)](https://pkg.go.dev/github.com/apache/dubbo-kubernetes)
[![Go Report Card](https://goreportcard.com/badge/github.com/apache/dubbo-kubernetes)](https://goreportcard.com/report/github.com/apache/dubbo-kubernetes)
[![codecov](https://codecov.io/gh/apache/dubbo-kubernetes/branch/master/graph/badge.svg)](https://codecov.io/gh/apache/dubbo-kubernetes)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

<p align="left">
  <a href="https://dubbo.apache.org">
    <img src="logo/dubbo-transparentbackground-unframed.svg" alt="Apache Dubbo" title="Apache Dubbo" width="240" height="320" >
  </a>
</p>

## Dubbo Service Mesh for Kubernetes

Implement an open-source service mesh for Dubbo in a Kubernetes cluster, integrate applications directly into the service mesh through the SDK, secure, connect, and observe services in a unified, lighter, and more efficient way, and support load balancing and service-to-service authentication.

## Project Core

- **api** — API definitions for Dubbo.
- **client-go** — Go client library for the Dubbo API.
- **dubboctl** — Command-line tool that provides control plane management, development framework setup, and application deployment capabilities.
- **dubbod** — The control plane, communicating based on gRPC and xDS APIs.
- **operator** — Provides user-friendly options for operating the service mesh.

## Quick Start

Please refer to [official website](https://cn.dubbo.apache.org/zh-cn/overview/mesh/)

## Contributing

Refer to [CONTRIBUTING.md](https://github.com/apache/dubbo-kubernetes/blob/master/CONTRIBUTING.md)

## License

Apache License 2.0, see [LICENSE](https://github.com/apache/dubbo-kubernetes/blob/master/LICENSE).
