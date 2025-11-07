# gRPC Proxyless Test Example

This is a test example for gRPC proxyless service mesh based on [Istio's blog post](https://istio.io/latest/blog/2021/proxyless-grpc/).

## Architecture

- **Consumer**: gRPC server with xDS support (port 17070)
- **Producer**: gRPC client with xDS support + test server (port 17171)

Both services use `dubbo-proxy` sidecar as an xDS proxy to connect to the control plane. The sidecar runs an xDS proxy server that listens on a Unix Domain Socket (UDS) at `/etc/dubbo/proxy/XDS`. The gRPC applications connect to this xDS proxy via the UDS socket using the `GRPC_XDS_BOOTSTRAP` environment variable.

**Note**: This is "proxyless" in the sense that the applications use native gRPC xDS clients instead of Envoy proxy for traffic routing. However, a lightweight sidecar (`dubbo-proxy`) is still used to proxy xDS API calls between the gRPC clients and the control plane.
