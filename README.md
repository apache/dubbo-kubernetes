<h1 align="center">
Dubbo Kubernetes
</h1>

[![Build](https://github.com/apache/dubbo-kubernetes/actions/workflows/ci.yml/badge.svg)](https://github.com/apache/dubbo-kubernetes/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/apache/dubbo-kubernetes/branch/master/graph/badge.svg)](https://codecov.io/gh/apache/dubbo-kubernetes)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

A cloud-native **proxyless service mesh** platform for Dubbo microservices, enabling zero-latency inter-service communication across multiple protocols, advanced traffic management, and enterprise-grade security without sidecar proxies. Built on Istio's control plane architecture with native Kubernetes integration, it delivers service mesh capabilities with minimal resource overhead and maximum performance.

## Project Structure

The main repositories of Dubbo on Kubernetes include:

- **dubboctl** — The command-line management tool that provides control plane management, development framework scaffolding, and application deployment.
- **dubbod** — The dubbo control plane. It is built on Istio to implement a proxyless service mesh and includes the following components:
  - **sail** - Runtime proxy configuration and XDS server
  - **aegis** - Certificate issuance and rotation
  - **gear** - Validation, aggregation, transformation, and distribution of Dubbo configuration
- **operator**: Provides user-friendly options to operate the Dubbo proxyless service mesh.

## Quick Start

Please refer to [official website](https://cn.dubbo.apache.org/zh-cn/overview/home/)

## Contributing

Refer to [CONTRIBUTING.md](https://github.com/apache/dubbo-kubernetes/blob/master/CONTRIBUTING.md)

## License

Apache License 2.0, see [LICENSE](https://github.com/apache/dubbo-kubernetes/blob/master/LICENSE).
