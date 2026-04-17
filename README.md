<h1 align="center">Apache Kdubbo</h1>

<p align="center">
  <a href="https://pkg.go.dev/github.com/apache/dubbo-kubernetes">
    <img src="https://pkg.go.dev/badge/github.com/apache/dubbo-kubernetes.svg" />
  </a>
  <a href="https://goreportcard.com/report/github.com/apache/dubbo-kubernetes">
    <img src="https://goreportcard.com/badge/github.com/apache/dubbo-kubernetes" />
  </a>
  <a href="https://codecov.io/gh/apache/dubbo-kubernetes">
    <img src="https://codecov.io/gh/apache/dubbo-kubernetes/branch/master/graph/badge.svg" />
  </a>
  <img src="https://img.shields.io/badge/license-Apache--2.0-green.svg" />
</p>

Dubbo inherent mesh implemented for the underlying cluster management platform can directly receive policies from the control plane and obtain features such as load balancing, service discovery, and observability without requiring a sidecar proxy.

## Introduction

> ⚠ Current version is in the **Alpha** phase. The `0.5.0` release will be the first **RC** phase.

Dubbo’s control plane provides an abstraction layer over the underlying cluster management platform.

Dubbo component composition:

- **dubbod** — Dubbo xDS control plane. It provides service discovery, configuration and certificate issuance.

- **dubbo-go-pixiu** — Used for ingress/egress traffic between services in the cluster and external services.

## Repositories

Projects are distributed across the code directory repositories:

- [dubbo/api](https://github.com/kdubbo/api). — Defines the component level APIs for the Dubbo control plane.

- [dubbo/xds-api](https://github.com/kdubbo/xds-api). — Define the xDS API for the Dubbo control plane.

- [dubbo/client-go](https://github.com/kdubbo/client-go). — Defines the Kubernetes clients automatically generated for Dubbo control plane resources.

- [dubbo/dubboctl](./dubboctl). — Provides command line tools for control plane management and other operations.

- [dubbo/dubbod](./dubbod) — The main code directory for the Dubbo control plane.

- [dubbo/operator](./operator). — Provides user friendly options for operating the service mesh.

## Contributing

Refer to [CONTRIBUTING.md](./CONTRIBUTING.md)

## License

Apache License 2.0, see [LICENSE](https://github.com/apache/dubbo-kubernetes/blob/master/LICENSE).
