# Dubbod Helm Chart

This chart installs an Base deployment.

## Setup Repo Info
```
helm repo add dubbo https://charts.dubbo.apache.org
helm repo update
```
See [helm repo](https://helm.sh/docs/helm/helm_repo/) for command documentation.

## Installing the Chart

To install the chart with the release name dubbo:
```
kubectl create namespace dubbo-system
helm install nacos dubbo/dubbod --namespace dubbo-system
```

## Uninstalling the Chart

To uninstall/delete the dubbo deployment:
```
helm delete dubbo --namespace dubbo-system
```

## Configuration

To view support configuration options and documentation, run:
```
helm show values dubbo/dubbod
```

### profiles
Dubbo Helm Chart introduces the concept of profiles, which are predefined sets of configuration values. You can specify the desired profile using --set profile=<profile>. For example, the demo profile provides a preset configuration suitable for testing environments, with additional features enabled and reduced resource requirements.

To maintain consistency, all charts support the same profiles, even if some settings don’t apply to a particular chart.

The precedence of values is as follows:
1. Explicitly set parameters (via --set)
2. Values defined in the selected profile
3. Default values of the chart

In actual configuration, you do not need to include the nested path under defaults. For example, you should use:
```
--set some.field=true
```

instead of:
```
--set defaults.some.field=true
```