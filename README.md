<h1 align="center">
Dubbo Service Mesh for Kubernetes
</h1>

[![Go Report Card](https://goreportcard.com/badge/github.com/apache/dubbo-kubernetes)](https://goreportcard.com/report/github.com/apache/dubbo-kubernetes)
[![Go Reference](https://pkg.go.dev/badge/github.com/apache/dubbo-kubernetes.svg)](https://pkg.go.dev/github.com/apache/dubbo-kubernetes)
[![Release](https://img.shields.io/github/release/apache/dubbo-kubernetes.svg)](https://github.com/apache/dubbo-kubernetes/releases)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

Cloud-Native proxyless service mesh platform for Dubbo microservices, enabling zero-latency inter-service communication across multiple protocols, advanced traffic management, and enterprise-grade security without sidecar proxies. Built on Istio's control plane architecture with native Kubernetes integration, it delivers service mesh capabilities with minimal resource overhead and maximum performance.

## Project Structure

The main repositories of Dubbo on Kubernetes include:

- **dubboctl** — The command-line management tool that provides control plane management, development framework scaffolding, and application deployment.
- **dubbod** — The dubbo control plane. It is similar to the Istio components, used to implement a proxyless service mesh, and includes the following components:
  - **planet** - Runtime proxy configuration and xDS server
  - **aegis** - Certificate issuance and rotation
  - **gear** - Validation, aggregation, transformation, and distribution of Dubbo configuration
- **operator** — Provides user-friendly options to operate the Dubbo proxyless service mesh.

## Quick Start

Please refer to [official website](https://cn.dubbo.apache.org/zh-cn/overview/home/)

## Contributing

Refer to [CONTRIBUTING.md](https://github.com/apache/dubbo-kubernetes/blob/master/CONTRIBUTING.md)

## License

Apache License 2.0, see [LICENSE](https://github.com/apache/dubbo-kubernetes/blob/master/LICENSE).
