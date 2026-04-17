# Dubbo Base Helm Chart

This chart installs an Dubbo Base deployment.

## Setup Repo Info

```bash
helm repo add dubbo https://charts.dubbo.apache.org
helm repo update
```
See [helm repo](https://helm.sh/docs/helm/helm_repo/) for command documentation.

## Installing the Chart

To install the chart with the release name dubbo:

```bash
kubectl create namespace dubbo-system
helm install base dubbo/base --namespace dubbo-system
```

## Uninstalling the Chart

To uninstall/delete the dubbo deployment:

```bash
helm delete dubbo --namespace dubbo-system
```