# gRPC Proxyless Example

This example demonstrates how to deploy gRPC applications with proxyless service mesh support using Dubbo Kubernetes.

## Overview

This sample includes:
- **Consumer**: A gRPC server that receives requests (port 17070)
- **Producer**: A gRPC client that sends requests to the consumer service

Both services use native gRPC xDS clients to connect to the Dubbo control plane through the `dubbo-proxy` sidecar, enabling service discovery, load balancing, and traffic management without requiring Envoy proxy for application traffic.

## Prerequisites

1. Kubernetes cluster with Dubbo Kubernetes control plane installed
2. `kubectl` configured to access the cluster
3. `grpcurl` (optional, for testing)

## Deployment

### 1. Create Namespace

```bash
kubectl create ns grpc-proxyless
kubectl label namespace grpc-proxyless dubbo-injection=enabled
```

### 2. Deploy Services

```bash
kubectl apply -f grpc-proxyless.yaml
```

### 3. Verify Deployment

```bash
# Check pods are running
kubectl get pods -n grpc-proxyless

# Check services
kubectl get svc -n grpc-proxyless
```

## Configuration

### Key Annotations

- `proxyless.dubbo.apache.org/inject: "true"` - Enables proxyless injection
- `inject.dubbo.apache.org/templates: grpc-agent` - Uses the grpc-agent template
- `proxy.dubbo.apache.org/config: '{"holdApplicationUntilProxyStarts": true}'` - Ensures proxy starts before application

### Environment Variables

The `dubbo-proxy` sidecar automatically:
- Creates `/etc/dubbo/proxy/grpc-bootstrap.json` bootstrap file
- Exposes xDS proxy via Unix Domain Socket
- Sets `GRPC_XDS_BOOTSTRAP` environment variable for application containers

## Testing

### Test with grpcurl

1. Port forward to producer service:

```bash
kubectl port-forward -n grpc-proxyless \
  $(kubectl get pod -l app=producer -n grpc-proxyless -o jsonpath='{.items[0].metadata.name}') \
  17171:17171 &
```

2. Send test request:

```bash
grpcurl -plaintext -d '{
  "url": "xds:///consumer.grpc-proxyless.svc.cluster.local:7070",
  "count": 5
}' localhost:17171 echo.EchoTestService/ForwardEcho
```

Expected output:
```json
{
  "output": [
    "[0 body] Hostname=consumer-xxx",
    "[1 body] Hostname=consumer-yyy",
    "[2 body] Hostname=consumer-xxx",
    "[3 body] Hostname=consumer-yyy",
    "[4 body] Hostname=consumer-xxx"
  ]
}
```

### Check Logs

```bash
# Consumer logs
kubectl logs -f -l app=consumer -n grpc-proxyless -c app

# Producer logs
kubectl logs -f -l app=producer -n grpc-proxyless -c app

# Proxy sidecar logs
kubectl logs -f -l app=consumer -n grpc-proxyless -c dubbo-proxy
```

## Troubleshooting

### Application fails to start

If the application fails with "grpc-bootstrap.json: no such file or directory":
- The `dubbo-proxy` sidecar may not have generated the bootstrap file yet
- Check proxy logs: `kubectl logs <pod-name> -c dubbo-proxy -n grpc-proxyless`
- Ensure `holdApplicationUntilProxyStarts: true` is set in annotations

### Connection issues

1. Verify xDS proxy is running:
```bash
kubectl exec <pod-name> -c dubbo-proxy -n grpc-proxyless -- ls -la /etc/dubbo/proxy/
```

2. Check bootstrap file:
```bash
kubectl exec <pod-name> -c app -n grpc-proxyless -- cat /etc/dubbo/proxy/grpc-bootstrap.json
```

3. Verify control plane connectivity:
```bash
kubectl logs <pod-name> -c dubbo-proxy -n grpc-proxyless | grep -i xds
```

## Cleanup

```bash
kubectl delete -f grpc-proxyless.yaml
kubectl delete ns grpc-proxyless
```
