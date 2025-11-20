# Dubbo Service Mesh for Kubernetes

[![Go Reference](https://pkg.go.dev/badge/github.com/apache/dubbo-kubernetes.svg)](https://pkg.go.dev/github.com/apache/dubbo-kubernetes)
[![Go Report Card](https://goreportcard.com/badge/github.com/apache/dubbo-kubernetes)](https://goreportcard.com/report/github.com/apache/dubbo-kubernetes)
[![codecov](https://codecov.io/gh/apache/dubbo-kubernetes/branch/master/graph/badge.svg)](https://codecov.io/gh/apache/dubbo-kubernetes)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

This project is an open-source proxyless service mesh for Dubbo microservices. Built on the Istio control plane architecture, it leverages its core capabilities and integrates deeply with Kubernetes, providing full service mesh functionality with minimal resource overhead, enabling high-performance inter-service communication, traffic management, and security features.

## Project Structure

The main repositories of Dubbo on Kubernetes include:

- **dubboctl** — The command-line management tool that provides control plane management, development framework scaffolding, and application deployment.
- **dubbod** — The dubbo control plane. used to implement a Dubbo proxyless service mesh.
- **operator** — Provides user-friendly options to operate the Dubbo proxyless service mesh.

## Quick Start

Please refer to [official website](https://cn.dubbo.apache.org/zh-cn/overview/home/)

## Contributing

Refer to [CONTRIBUTING.md](https://github.com/apache/dubbo-kubernetes/blob/master/CONTRIBUTING.md)

## License

Apache License 2.0, see [LICENSE](https://github.com/apache/dubbo-kubernetes/blob/master/LICENSE).
