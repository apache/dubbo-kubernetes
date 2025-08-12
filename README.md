<h1 align="center">
Dubbo Kubernetes
</h1>

[![Build](https://github.com/apache/dubbo-kubernetes/actions/workflows/ci.yml/badge.svg)](https://github.com/apache/dubbo-kubernetes/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/apache/dubbo-kubernetes/branch/master/graph/badge.svg)](https://codecov.io/gh/apache/dubbo-kubernetes)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

Provides support for building and deploying Dubbo applications in various environments, including Kubernetes and Alibaba Cloud ACK.

## Repositories
The main code repositories of Dubbo on Kubernetes include:

- [dubboctl](dubboctl/): This directory contains code for the command line utility.
- [helm-charts](manifests/charts): This directory contains the [Helm chart](https://github.com/apache/dubbo-helm-charts) sources, which are versioned, built, and pushed to the following Helm repositories with each Dubbo release.
- dubbod — The dubbo control plane. It is built on Istio to implement a proxyless service mesh and includes the following components:
  - [navigator](navigator/) (under development): Responsible for configuring proxies at runtime.
- [Operator](operator/): Dubbo operator provides user friendly options to operate the Dubbo proxyless mesh.

## Quick Start
Please refer to [official website](https://cn.dubbo.apache.org/zh-cn/overview/home/)

## Contributing

Refer to [CONTRIBUTING.md](https://github.com/apache/dubbo-kubernetes/blob/master/CONTRIBUTING.md)

## License

Apache License 2.0, see [LICENSE](https://github.com/apache/dubbo-kubernetes/blob/master/LICENSE).