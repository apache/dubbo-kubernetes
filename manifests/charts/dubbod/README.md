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
  --set-string global.inherent.cni.image=registry.example.com/dubbod:${VERSION}
```

## Security configuration

The built-in rotating CA and TLS 1.2 minimum are enabled by default. A mounted
four-file CA Secret (`ca-cert.pem`, `ca-key.pem`, `cert-chain.pem`, and
`root-cert.pem`) can replace it:

```bash
helm upgrade --install dubbod ./manifests/charts/dubbod \
  --namespace dubbo-system \
  --set security.ca.provider=plugin \
  --set security.ca.plugin.secretName=cacerts
```

To delegate certificate signing to the Kubernetes CSR API, provide the signer,
its root bundle ConfigMap, and a matching `meshConfig.caCertificates` entry:

```bash
helm upgrade --install dubbod ./manifests/charts/dubbod \
  --namespace dubbo-system \
  --set security.ca.provider=kubernetes \
  --set-string security.ca.kubernetes.signerName=example.com/dubbo \
  --set-string security.ca.kubernetes.rootConfigMapName=dubbo-ca-root-cert
```

Set `meshConfig.minimumTlsVersion=TLSV1_3` to require TLS 1.3. Configure named
HTTP or gRPC external authorization services under
`meshConfig.extensionProviders`; `AuthorizationPolicy` resources reference
them through `provider.name`.

## Uninstalling the Chart

To uninstall/delete the dubbo deployment:

```bash
helm delete dubbod --namespace dubbo-system
```
