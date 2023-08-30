<h1 align="center">
The Dubbo Kubernetes Integration
</h1>

<p align="center" style="color: red; font-weight: bold;">
⚠️ This is still an experimental version. ⚠️
</p>

[![Build](https://github.com/apache/dubbo-kubernetes/actions/workflows/ci.yml/badge.svg)](https://github.com/apache/dubbo-kubernetes/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/apache/dubbo-kubernetes/branch/master/graph/badge.svg)](https://codecov.io/gh/apache/dubbo-kubernetes)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

This repository contains libraries and tools for creating and deploying Dubbo applications in any Kubernetes environment, i.e. on Kubernetes, Aliyun ACK, etc.

## Prerequisites:
* Ensure you have Go installed, version 1.20 or higher.
* Make sure you install kubectl.
* Ensure you have Dubboctl installed.

## Quick Start
### Create a Dubbo application
Use `dubboctl create` to create a project template.

```shell
dubboctl create -l java
```

This should generate a simple project with a demo service properly configured and is ready to run. 

> For java developers, it's recommended to use [start.dubbo.apache.org]() or [IntelliJ IDEA plugin]() to generate more complicated templates.

### Deploy application to Kubernetes
Before deploying the application, let's install Nacos, Zookeeper, Prometheus and other components necessary for running a Dubbo application or microservice cluster.

```shell
dubboctl install --profile=demo # This will install Nacos, Prometheus, Grafana, Admin, etc.
```

Next, build your application as docker image and deploy it into kubernetes cluster with `dubboctl deploy`, it will do the following two steps:

1. Build your application from source code into docker image and push the image to remote repository.
2. Generate all the kubernetes configurations (e.g., deployments, services, load balancers) needed to run your application on vanilla Kubernetes.

```shell
dubboctl deploy --out=deployment.yml
```

Finally, apply manifests into kubernetes cluster.

```shell
kubectl apply -f deployment.yml
```

### Monitor and manage your application
We already have the application up and running, now it's time to continuously monitor the status or manage the traffics of our applications.

#### Admin
Run the following command and open `http://localhost:38080/admin/` with your favourite browser.

```shell
dubboctl dashboard admin
```

![Admin Console]()


![Admin Grafana]()

#### Tracing
```shell
dubboctl dashboard zipkin
```

#### Traffic Management
Please refer to our official website to learn the traffic policies in Dubbo with some well-designed tasks.
* Timeout
* Accesslog
* Region-aware traffic split
* Weight-based traffic split
* Circuit breaking
* Canary release




