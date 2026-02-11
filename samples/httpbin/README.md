# Httpbin Service

```bash
kubectl get crd gateways.gateway.networking.k8s.io &> /dev/null || \
{ kubectl kustomize "github.com/kubernetes-sigs/gateway-api/config/crd?ref=v1.4.0" | kubectl apply -f -; }
```

```yaml
kubectl create namespace dubbo-ingress
kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: gateway
  namespace: dubbo-ingress
spec:
  gatewayClassName: dubbo
  listeners:
  - name: default
    hostname: "*.example.com"
    port: 80
    protocol: HTTP
    allowedRoutes:
      namespaces:
        from: All
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: http
  namespace: default
spec:
  parentRefs:
  - name: gateway
    namespace: dubbo-ingress
  hostnames: ["httpbin.example.com"]
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /get
    backendRefs:
    - name: httpbin
      port: 8000
EOF
```

```bash
NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')
SVC_PORT=$(kubectl get svc gateway-dubbo -n dubbo-ingress -o jsonpath='{.spec.ports[?(@.port==80)].nodePort}')
curl -s -I -HHost:httpbin.example.com "http://$NODE_IP:$SVC_PORT/get"
```

```bash
kubectl delete httproute http
kubectl delete gateways.gateway.networking.k8s.io gateway -n dubbo-ingress
kubectl delete ns dubbo-ingress
```