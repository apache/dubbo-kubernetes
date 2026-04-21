# App Sample

## Install

```bash
kubectl create ns app
kubectl label namespace app dubbo-injection=enabled
kubectl apply -f samples/app/deployment.yaml
```

```bash
kubectl -n app patch deployment nginx --type='merge' -p '{"spec":{"template":{"metadata":{"labels":{"version":"v1","proxyless.dubbo.apache.org/inject":"true"},"annotations":{"proxyless.dubbo.apache.org/inject":"true","inject.dubbo.apache.org/templates":"grpc-engine"}}}}}'
kubectl -n app rollout restart deploy/nginx
kubectl -n app rollout status deploy/nginx --timeout=180s
```

## Uninstall

```bash
kubectl delete -f samples/app/deployment.yaml --ignore-not-found=true
kubectl delete ns app
```
