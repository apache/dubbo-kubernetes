# Apache Kdubbo - dubbo-mesh

[![Go Reference](https://pkg.go.dev/badge/github.com/apache/dubbo-kubernetes.svg)](https://pkg.go.dev/github.com/apache/dubbo-kubernetes)
[![Go Report Card](https://goreportcard.com/badge/github.com/apache/dubbo-kubernetes)](https://goreportcard.com/report/github.com/apache/dubbo-kubernetes)
[![License](https://img.shields.io/badge/license-Apache--2.0-green.svg)](https://www.apache.org/licenses/LICENSE-2.0)

Dubbo service mesh enables workloads to join natively, receive policies from the control plane via xDS, and gain capabilities such as load balancing, service discovery, and observability.

## Introduction

> [!WARNING]
> The current version is in the **Alpha** phase.
> 
> Releases `0.4.6–0.4.9` will be in the **Beta** phase.
> 
> Release `0.5.0` will be the first **RC** version.

Dubbo’s control plane provides an abstraction layer over the underlying cluster management platform.

Dubbo component composition:

- **dubbod** — Dubbo xDS control plane. It provides service discovery, configuration and certificate issuance.
- **dxplane** — Dubbo inbound mTLS terminator. It runs beside the workload to accept mesh traffic and forward it locally; outbound routing stays proxyless in the SDK.
- **dxgate** — Dubbo delegated gateway for Gateway API. It consumes routing configuration from dubbod over xDS and proxies north-south traffic into the mesh.

## Repositories

Projects are distributed across the code directory repositories:

- [api](https://github.com/kdubbo/api). — Defines the component level APIs for the Dubbo control plane.

- [xds-api](https://github.com/kdubbo/xds-api). — Define the xDS API for the Dubbo control plane.

- [client-go](https://github.com/kdubbo/client-go). — Defines the Kubernetes clients automatically generated for Dubbo control plane resources.

- [dubboctl](./cli). — Provides dubboctl command line tools for control plane management and other operations.

- [dubbod](./dubbod) — The main code directory for the Dubbo control plane.

- [operator](./operator). — Provides user friendly options for operating the service mesh.

- [dxgate](https://github.com/kdubbo/dxgate) — Provides the delegated gateway that serves Gateway API traffic at the mesh edge.

- [dxplane](https://github.com/kdubbo/dxplane) — Provides the inbound mTLS terminator that accepts mesh traffic on the workload's behalf.

- [gui](https://github.com/kdubbo/gui) — Provides the console that aggregates the management API across discovered control planes.

## Contributing

Refer to [CONTRIBUTING.md](./CONTRIBUTING.md)

## License

Apache License 2.0, see [LICENSE](https://github.com/apache/dubbo-kubernetes/blob/master/LICENSE).
