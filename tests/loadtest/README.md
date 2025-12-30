# Load Test Guide

## Test Modes

- **Baseline mode**: Can be tested directly (no control plane required)
- **Envoy mode**: Requires Istio control plane
- **xDS mode**: Requires Dubbo control plane (dubbod)

## Quick Start

### 1. Create Namespace

```bash
kubectl create ns loadtest
```

### 2. Configure and Deploy

Edit `k8s/loadtest.yaml` to set the desired mode, then apply:

```bash
kubectl apply -f k8s/loadtest.yaml
```

#### Baseline Mode

No annotations needed. Set mode to `baseline` in both server and client.

#### Envoy Mode

Requires Istio control plane. Add annotation and set mode:

```yaml
annotations:
  sidecar.istio.io/inject: "true"
```

Set mode to `envoy` in both server and client.

#### xDS Mode

Requires Dubbo control plane. Add annotations and set mode:

```yaml
annotations:
  proxyless.dubbo.apache.org/inject: "true"
  inject.dubbo.apache.org/templates: grpc-agent
```

Set mode to `xds` in both server and client. Client target should use `xds:///` scheme.

### 3. View Results

```bash
# Wait for server ready
kubectl rollout status deployment/loadtest-server -n loadtest --timeout=120s

# View server logs
kubectl logs -l app=loadtest-server -n loadtest -f

# View client logs (test results)
kubectl logs -l app=loadtest-client -n loadtest -f
```

## Verify Injection

Check pod containers:

```bash
kubectl get pod -l app=loadtest-server -n loadtest -o jsonpath='{.items[0].spec.containers[*].name}'
```

- **Baseline**: `server` (1 container)
- **Envoy**: `server istio-proxy` (2 containers)
- **xDS**: `server dubbo-proxy` (2 containers)

## Test Configuration

Default parameters in `k8s/loadtest.yaml`:
- QPS: 100 requests/second
- Duration: 60s
- Connections: 4
- Payload: 0 bytes

## Common Issues

**Injection not working**: Verify control plane is installed and MutatingWebhookConfiguration exists.

**Connection refused**: Check server pod status and logs.

**xDS connection timeout**: Verify dubbo-proxy sidecar is running and bootstrap file exists at `/etc/dubbo/proxy/grpc-bootstrap.json`.

## Cleanup

```bash
kubectl delete -f k8s/loadtest.yaml
```

## Results

See [RESULTS.md](RESULTS.md) for detailed performance comparison.
