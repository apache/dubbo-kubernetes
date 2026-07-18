# Dubbod Helm Chart

This chart installs the Dubbod deployment.

## Installing a Versioned Release

```bash
# Select a release whose Assets include the packaged charts.
VERSION="<version>"
kubectl create namespace dubbo-system
helm upgrade --install dubbo-base \
  "https://github.com/apache/dubbo-kubernetes/releases/download/${VERSION}/base-${VERSION}.tgz" \
  --namespace dubbo-system
helm upgrade --install dubbod \
  "https://github.com/apache/dubbo-kubernetes/releases/download/${VERSION}/dubbod-${VERSION}.tgz" \
  --namespace dubbo-system
```

Release `0.4.3` predates packaged chart assets; the commands above apply to
releases produced by the current release workflow.

The packaged chart defaults to
`ghcr.io/apache/dubbo-kubernetes/dubbod:${VERSION}`. Override the shared
control-plane/CNI image when using a mirror or a locally loaded image:

```bash
helm upgrade --install dubbod ./dubbod-${VERSION}.tgz \
  --namespace dubbo-system \
  --set-string global.proxyless.cni.image=registry.example.com/dubbod:${VERSION}
```

## Uninstalling the Chart

To uninstall/delete the dubbo deployment:

```bash
helm delete dubbod --namespace dubbo-system
```
