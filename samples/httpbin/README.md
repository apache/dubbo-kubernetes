# Httpbin 南北向流量示例

这个示例部署默认命名空间的 `httpbin`，并通过共享的 `dxgate-gateway` + `HTTPRoute` 触发 dubbod 托管的固定 dxgate 网关。

```bash
kubectl apply -f samples/httpbin/httpbin.yaml
kubectl get gateway dxgate-gateway
kubectl get svc dxgate-gateway
```

访问 `default/dxgate-gateway` 的外部地址即可进入 dxgate，再转发到 `httpbin.default.svc.cluster.local:8000`。

后续新增 `foo`、`bar` 这类服务时，继续创建 `HTTPRoute` 并把 `parentRefs.name` 指向 `dxgate-gateway`，不要为每个业务服务再创建一个新的 `Gateway`。
