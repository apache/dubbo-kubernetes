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

### 3. Call the test service

```bash
kubectl port-forward -n grpc-app $(kubectl get pod -l app=consumer -n grpc-app -o jsonpath='{.items[0].metadata.name}') 17171:17171
```

```bash
grpcurl -plaintext -d '{"url": "xds:///producer.grpc-app.svc.cluster.local:7070","count": 5}' localhost:17171 echo.EchoTestService/ForwardEcho
```

```json
{
  "output": [
    "[0 body] Hostname=producer-v2-594b6977c8-5gw2z ServiceVersion=v2 Namespace=grpc-app IP=192.168.219.88 ServicePort=17070",
    "[1 body] Hostname=producer-v1-fbb7b9bd9-l8frj ServiceVersion=v1 Namespace=grpc-app IP=192.168.219.119 ServicePort=17070",
    "[2 body] Hostname=producer-v2-594b6977c8-5gw2z ServiceVersion=v2 Namespace=grpc-app IP=192.168.219.88 ServicePort=17070",
    "[3 body] Hostname=producer-v1-fbb7b9bd9-l8frj ServiceVersion=v1 Namespace=grpc-app IP=192.168.219.119 ServicePort=17070",
    "[4 body] Hostname=producer-v2-594b6977c8-5gw2z ServiceVersion=v2 Namespace=grpc-app IP=192.168.219.88 ServicePort=17070"
  ]
}
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

Using the subsets defined above, you can send weighted traffic to different versions. The following example sends 10% of traffic to v1 and 90% to v2:

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
      weight: 10
    - destination:
        host: producer.grpc-app.svc.cluster.local
        subset: v2
      weight: 90
EOF
```

Now, send a set of 10 requests to verify the traffic distribution:

```bash
grpcurl -plaintext -d '{"url": "xds:///producer.grpc-app.svc.cluster.local:7070","count": 5}' localhost:17171 echo.EchoTestService/ForwardEcho
```

The response should contain mostly `v2` responses, demonstrating the weighted traffic splitting:

```json
{
  "output": [
    "[0 body] Hostname=producer-v2-594b6977c8-5gw2z ServiceVersion=v2 Namespace=grpc-app IP=192.168.219.88 ServicePort=17070",
    "[1 body] Hostname=producer-v2-594b6977c8-5gw2z ServiceVersion=v2 Namespace=grpc-app IP=192.168.219.88 ServicePort=17070",
    "[2 body] Hostname=producer-v2-594b6977c8-5gw2z ServiceVersion=v2 Namespace=grpc-app IP=192.168.219.88 ServicePort=17070",
    "[3 body] Hostname=producer-v1-fbb7b9bd9-l8frj ServiceVersion=v1 Namespace=grpc-app IP=192.168.219.119 ServicePort=17070",
    "[4 body] Hostname=producer-v2-594b6977c8-5gw2z ServiceVersion=v2 Namespace=grpc-app IP=192.168.219.88 ServicePort=17070"
  ]
}
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
grpcurl -plaintext -d '{"url": "xds:///producer.grpc-app.svc.cluster.local:7070","count": 5}' localhost:17171 echo.EchoTestService/ForwardEcho
```

Expected error output:
```json
{
   "output": [
      "ERROR:\nCode: Unknown\nMessage: 5/5 requests had errors; first error: rpc error: code = Unavailable desc = connection error: desc = \"transport: authentication handshake failed: tls: first record does not look like a TLS handshake\"",
      "[0] Error: rpc error: code = Unavailable desc = connection error: desc = \"transport: authentication handshake failed: tls: first record does not look like a TLS handshake\"",
      "[1] Error: rpc error: code = Unavailable desc = connection error: desc = \"transport: authentication handshake failed: tls: first record does not look like a TLS handshake\"",
      "[2] Error: rpc error: code = Unavailable desc = connection error: desc = \"transport: authentication handshake failed: tls: first record does not look like a TLS handshake\"",
      "[3] Error: rpc error: code = Unavailable desc = connection error: desc = \"transport: authentication handshake failed: tls: first record does not look like a TLS handshake\"",
      "[4] Error: rpc error: code = Unavailable desc = connection error: desc = \"transport: authentication handshake failed: tls: first record does not look like a TLS handshake\""
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
grpcurl -plaintext -d '{"url": "xds:///producer.grpc-app.svc.cluster.local:7070","count": 5}' localhost:17171 echo.EchoTestService/ForwardEcho
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
