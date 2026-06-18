# Apache Kdubbo (dubbo-mesh)

[![Go Reference](https://pkg.go.dev/badge/github.com/apache/dubbo-kubernetes.svg)](https://pkg.go.dev/github.com/apache/dubbo-kubernetes)
[![Go Report Card](https://goreportcard.com/badge/github.com/apache/dubbo-kubernetes)](https://goreportcard.com/report/github.com/apache/dubbo-kubernetes)
[![Codecov](https://codecov.io/gh/apache/dubbo-kubernetes/branch/master/graph/badge.svg)](https://codecov.io/gh/apache/dubbo-kubernetes)
[![License](https://img.shields.io/badge/license-Apache--2.0-green.svg)](https://www.apache.org/licenses/LICENSE-2.0)

Dubbo inherent mesh implemented for the underlying cluster management platform can directly receive policies from the control plane and obtain features such as load balancing, service discovery, and observability without requiring a sidecar proxy.

## Introduction

> [!WARNING]
> Current version is in the **Alpha** phase. The `0.5.0` release will be the first **RC** phase.
> 
Dubbo’s control plane provides an abstraction layer over the underlying cluster management platform.

Dubbo component composition:

- **dubbod** — Dubbo xDS control plane. It provides service discovery, configuration and certificate issuance.
- **dxgate** — Dubbo delegated gateway for Gateway API.

## Repositories

Projects are distributed across the code directory repositories:

- [dubbo/api](https://github.com/kdubbo/api). — Defines the component level APIs for the Dubbo control plane.

- [dubbo/xds-api](https://github.com/kdubbo/xds-api). — Define the xDS API for the Dubbo control plane.

- [dubbo/client-go](https://github.com/kdubbo/client-go). — Defines the Kubernetes clients automatically generated for Dubbo control plane resources.

- [dubbo/cli](./cli). — Provides dubboctl command line tools for control plane management and other operations.

- [dubbo/dubbod](./dubbod) — The main code directory for the Dubbo control plane.

- [dubbo/operator](./operator). — Provides user friendly options for operating the service mesh.

## Contributing

Refer to [CONTRIBUTING.md](./CONTRIBUTING.md)

## License

Apache License 2.0, see [LICENSE](https://github.com/apache/dubbo-kubernetes/blob/master/LICENSE).
