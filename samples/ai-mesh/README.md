# Mesh-native AI services

This sample uses one API: `networking.dubbo.apache.org/v1alpha3`
`DxgateService`. Ordinary HTTP backends remain core Kubernetes `Service`
objects. `dubbod` compiles both kinds of `HTTPRoute` backend into RDS and
delivers it to dxgate over xDS.

```bash
kubectl create namespace ai-mesh
kubectl -n ai-mesh apply -f gateway.yaml
kubectl -n ai-mesh apply -f backends.yaml
kubectl -n ai-mesh apply -f services.yaml
kubectl -n ai-mesh apply -f routes.yaml
```

The sample image `kdubbo/agent-mock:latest` is the no-key E2E fixture in
`tests/e2e/agentmock`; replace its Services and Secret values with production
backends and credentials.
