# App Sample

## Install

```bash
kubectl create ns app --dry-run=client -o yaml | kubectl apply -f -
kubectl label namespace app dubbo-injection=enabled --overwrite
kubectl apply -f samples/app/deployment.yaml
kubectl -n app rollout status deploy/nginx-v1 --timeout=180s
kubectl -n app rollout status deploy/nginx-v2 --timeout=180s
kubectl -n app rollout status deploy/nginx-consumer --timeout=180s
kubectl -n app get po --show-labels
```

## Traffic Rules

```bash
cat <<EOF | kubectl apply -f -
apiVersion: networking.dubbo.apache.org/v1alpha3
kind: DestinationRule
metadata:
  name: nginx-versions
  namespace: app
spec:
  host: nginx.app.svc.cluster.local
  subsets:
  - name: v1
    labels:
      version: v1
  - name: v2
    labels:
      version: v2
EOF

cat <<EOF | kubectl apply -f -
apiVersion: networking.dubbo.apache.org/v1alpha3
kind: VirtualService
metadata:
  name: nginx-weights
  namespace: app
spec:
  hosts:
  - nginx.app.svc.cluster.local
  http:
  - route:
    - destination:
        host: nginx.app.svc.cluster.local
        subset: v1
      weight: 10
    - destination:
        host: nginx.app.svc.cluster.local
        subset: v2
      weight: 90
EOF
```

## Consumer Bootstrap Test

```bash
kubectl -n app exec deploy/nginx-consumer -- sh -c \
  'echo GRPC_XDS_BOOTSTRAP=$GRPC_XDS_BOOTSTRAP; echo DUBBO_GRPC_XDS_CONFIG=$DUBBO_GRPC_XDS_CONFIG; test -s /etc/dubbo/proxy/grpc-bootstrap.json && echo bootstrap-present; test -s /etc/dubbo/proxy/dubbo-grpc-xds.json && echo runtime-config-present'
```

Expected output includes:

```text
GRPC_XDS_BOOTSTRAP=/etc/dubbo/proxy/grpc-bootstrap.json
DUBBO_GRPC_XDS_CONFIG=/etc/dubbo/proxy/dubbo-grpc-xds.json
bootstrap-present
runtime-config-present
```

## Runtime Config Test

```bash
kubectl -n app exec deploy/nginx-consumer -- python /client/request.py 100 | sort | uniq -c
```

Expected output should be close to the configured 10/90 split:

```text
  10 nginx v1
  90 nginx v2
```

The consumer reads `DUBBO_GRPC_XDS_CONFIG=/etc/dubbo/proxy/dubbo-grpc-xds.json`, selects endpoints and weights from the control-plane generated runtime config, and calls provider Pod IPs directly. It does not rely on Envoy, Istio, or grpc-go xDS.

## Uninstall

```bash
kubectl -n app delete destinationrule nginx-versions --ignore-not-found=true
kubectl -n app delete virtualservice nginx-weights --ignore-not-found=true
kubectl delete -f samples/app/deployment.yaml --ignore-not-found=true
kubectl delete ns app
```
