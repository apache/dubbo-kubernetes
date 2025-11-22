# gRPC Proxyless Example

This example demonstrates how to deploy gRPC applications with proxyless service mesh support using Dubbo Kubernetes.

## Overview

This sample includes:
- **Producer**: A gRPC server that receives requests (port 17070) and is deployed with multiple versions (v1/v2) to showcase gray release scenarios.
- **Consumer**: A gRPC client that sends requests to the producer service and exposes a test server (port 17171) for driving traffic via `grpcurl`.

Both services use native gRPC xDS clients to connect to the Dubbo control plane through the `dubbo-proxy` sidecar, enabling service discovery, load balancing, and traffic management without requiring Envoy proxy for application traffic.

## Prerequisites

1. Kubernetes cluster with Dubbo Kubernetes control plane installed
2. `kubectl` configured to access the cluster
3. `grpcurl` (optional, for testing)

## Deployment

### 1. Create Namespace

```bash
kubectl create ns grpc-app
kubectl label namespace grpc-app dubbo-injection=enabled
```

### 2. Deploy Services

```bash
kubectl apply -f grpc-app.yaml
```

### 3. Verify Deployment

```bash
# Check pods are running
kubectl get pods -n grpc-app

# Check services
kubectl get svc -n grpc-app
```

## Configuration

### Key Annotations

- `proxyless.dubbo.apache.org/inject: "true"` - Enables proxyless injection
- `inject.dubbo.apache.org/templates: grpc-agent` - Uses the grpc-agent template
- `proxy.dubbo.apache.org/config: '{"holdApplicationUntilProxyStarts": true}'` - Ensures proxy starts before application

### Security requirements

When mTLS is enabled (`SubsetRule` with `ISTIO_MUTUAL` or `PeerAuthentication STRICT`), **both the producer and consumer application containers must**:

1. Mount the certificate output directory that `dubbo-proxy` writes to:
   ```yaml
   volumeMounts:
     - name: dubbo-data
       mountPath: /var/lib/dubbo/data
   ```
   The `grpc-agent` template already provides the `dubbo-data` volume; mounting it from the app container makes the generated `cert-chain.pem`, `key.pem`, and `root-cert.pem` visible to gRPC.

2. Set `GRPC_XDS_EXPERIMENTAL_SECURITY_SUPPORT=true` so the gRPC runtime actually consumes the xDS security config instead of falling back to plaintext. The sample manifest in `grpc-app.yaml` shows the required environment variable.

3. When testing with `grpcurl`, export the same variable before issuing TLS requests:
   ```bash
   export GRPC_XDS_EXPERIMENTAL_SECURITY_SUPPORT=true
   grpcurl -d '{"url":"xds:///producer.grpc-app.svc.cluster.local:7070","count":5}' localhost:17171 echo.EchoTestService/ForwardEcho
   ```

## Testing

### Test with grpcurl

1. Port forward to the consumer test server:

```bash
kubectl port-forward -n grpc-app \
  $(kubectl get pod -l app=consumer -n grpc-app -o jsonpath='{.items[0].metadata.name}') \
  17171:17171 &
```

2. Send test request:

```bash
grpcurl -plaintext -d '{
  "url": "xds:///producer.grpc-app.svc.cluster.local:7070",
  "count": 5
}' localhost:17171 echo.EchoTestService/ForwardEcho
```

Expected output:
```json
{
  "output": [
    "[0 body] Hostname=producer-xxx",
    "[1 body] Hostname=producer-yyy",
    "[2 body] Hostname=producer-xxx",
    "[3 body] Hostname=producer-yyy",
    "[4 body] Hostname=producer-xxx"
  ]
}
```

### Check Logs

```bash
# Producer logs
kubectl logs -f -l app=producer -n grpc-app -c app

# Consumer logs
kubectl logs -f -l app=consumer -n grpc-app -c app

# Proxy sidecar logs
kubectl logs -f -l app=producer -n grpc-app -c dubbo-proxy
```

## Troubleshooting

### Application fails to start

If the application fails with "grpc-bootstrap.json: no such file or directory":
- The `dubbo-proxy` sidecar may not have generated the bootstrap file yet
- Check proxy logs: `kubectl logs <pod-name> -c dubbo-proxy -n grpc-app`
- Ensure `holdApplicationUntilProxyStarts: true` is set in annotations

### Connection issues

1. Verify xDS proxy is running:
```bash
kubectl exec <pod-name> -c dubbo-proxy -n grpc-app -- ls -la /etc/dubbo/proxy/
```

2. Check bootstrap file:
```bash
kubectl exec <pod-name> -c app -n grpc-app -- cat /etc/dubbo/proxy/grpc-bootstrap.json
```

3. Verify control plane connectivity:
```bash
kubectl logs <pod-name> -c dubbo-proxy -n grpc-app | grep -i xds
```

## Traffic Management

### Creating subsets with SubsetRule

First, create a subset for each version of the workload to enable traffic splitting:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: networking.dubbo.apache.org/v1
kind: SubsetRule
metadata:
  name: producer-versions
  namespace: grpc-app
spec:
  host: producer.grpc-app.svc.cluster.local
  subsets:
  - name: v1
    labels:
      version: v1
  - name: v2
    labels:
      version: v2
EOF
```

### Traffic shifting

Using the subsets defined above, you can send weighted traffic to different versions. The following example sends 20% of traffic to v1 and 80% to v2:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: networking.dubbo.apache.org/v1
kind: ServiceRoute
metadata:
  name: producer-weights
  namespace: grpc-app
spec:
  hosts:
  - producer.grpc-app.svc.cluster.local
  http:
  - route:
    - destination:
        host: producer.grpc-app.svc.cluster.local
        subset: v1
      weight: 20
    - destination:
        host: producer.grpc-app.svc.cluster.local
        subset: v2
      weight: 80
EOF
```

Now, send a set of 10 requests to verify the traffic distribution:

```bash
grpcurl -plaintext -d '{"url": "xds:///producer.grpc-app.svc.cluster.local:7070", "count": 10}' localhost:17171 echo.EchoTestService/ForwardEcho | jq -r '.output | join("")' | grep ServiceVersion
```

The response should contain mostly `v2` responses, demonstrating the weighted traffic splitting:

```plain
[0 body] ServiceVersion=v2
[1 body] ServiceVersion=v2
[2 body] ServiceVersion=v1
[3 body] ServiceVersion=v2
[4 body] ServiceVersion=v1
[5 body] ServiceVersion=v2
[6 body] ServiceVersion=v2
[7 body] ServiceVersion=v2
[8 body] ServiceVersion=v2
[9 body] ServiceVersion=v2
```

## Enabling mTLS

Due to the changes to the application itself required to enable security in gRPC, Dubbo Kubernetes's traditional method of automatically detecting mTLS support is unreliable. For this reason, the initial release requires explicitly enabling mTLS on both the client and server.

### Enable client-side mTLS

To enable client-side mTLS, apply a `SubsetRule` with `tls` settings:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: networking.dubbo.apache.org/v1
kind: SubsetRule
metadata:
  name: producer-mtls
  namespace: grpc-app
spec:
  host: producer.grpc-app.svc.cluster.local
  trafficPolicy:
    tls:
      mode: ISTIO_MUTUAL
EOF
```

Now an attempt to call the server that is not yet configured for mTLS will fail:

```bash
grpcurl -plaintext -d '{"url": "xds:///producer.grpc-app.svc.cluster.local:7070"}' localhost:17171 echo.EchoTestService/ForwardEcho | jq -r '.output | join("")'
```

Expected error output:
```json
{
  "output": [
    "ERROR:\nCode: Unknown\nMessage: 1/1 requests had errors; first error: rpc error: code = Unavailable desc = all SubConns are in TransientFailure"
  ]
}
```

### Enable server-side mTLS

To enable server-side mTLS, apply a `PeerAuthentication` policy. The following policy forces STRICT mTLS for the entire namespace:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: security.dubbo.apache.org/v1
kind: PeerAuthentication
metadata:
  name: producer-mtls
  namespace: grpc-app
spec:
  mtls:
    mode: STRICT
EOF
```

Requests will start to succeed after applying the policy:

```bash
grpcurl -plaintext -d '{"url": "xds:///producer.grpc-app.svc.cluster.local:7070"}' localhost:17171 echo.EchoTestService/ForwardEcho | jq -r '.output | join("")'
```

Expected successful output:
```json
{
  "output": [
    "[0 body] Hostname=producer-v2-594b6977c8-5gw2z ServiceVersion=v2 Namespace=grpc-app IP=192.168.219.88 ServicePort=17070",
    "[1 body] Hostname=producer-v2-594b6977c8-5gw2z ServiceVersion=v2 Namespace=grpc-app IP=192.168.219.88 ServicePort=17070",
    "[2 body] Hostname=producer-v2-594b6977c8-5gw2z ServiceVersion=v2 Namespace=grpc-app IP=192.168.219.88 ServicePort=17070",
    "[3 body] Hostname=producer-v2-594b6977c8-5gw2z ServiceVersion=v2 Namespace=grpc-app IP=192.168.219.88 ServicePort=17070",
    "[4 body] Hostname=producer-v1-fbb7b9bd9-l8frj ServiceVersion=v1 Namespace=grpc-app IP=192.168.219.119 ServicePort=17070"
  ]
}
```

## Cleanup

```bash
kubectl delete -f grpc-app.yaml
kubectl delete ns grpc-app
```
