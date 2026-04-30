# Httpbin 南北向流量示例

这个示例部署默认命名空间的 `httpbin`，并通过 `Gateway` + `HTTPRoute` 触发 dubbod 托管的固定 dxgate 网关。

```bash
kubectl apply -f samples/httpbin/httpbin.yaml
kubectl get gateway httpbin-gateway
kubectl -n dubbo-system get svc dxgate
```

访问 `dubbo-system/dxgate` 的外部地址即可进入 dxgate，再转发到 `httpbin.default.svc.cluster.local:8000`。
