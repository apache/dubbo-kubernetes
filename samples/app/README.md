# 示例说明

## 准备

先构建并更新 `kdubbo/dubbod:debug` 镜像，再执行下面命令。

## 安装

```bash
kubectl create ns app --dry-run=client -o yaml | kubectl apply -f -
kubectl label namespace app dubbo-injection=enabled --overwrite
kubectl apply -f samples/app/deployment.yaml
kubectl -n app rollout status deploy/nginx-v1 --timeout=180s
kubectl -n app rollout status deploy/nginx-v2 --timeout=180s
kubectl -n app rollout status deploy/nginx-consumer --timeout=180s
```

`dubbo-injection=enabled` 开启后，会自动注入 `grpc-engine`。`nginx-consumer` 使用 `kdubbo/dubbod` 镜像，会自动启动 `xclient --watch`。

## 配置流量规则

```bash
kubectl -n app delete virtualservice nginx-weights --ignore-not-found=true
kubectl -n app delete destinationrule nginx-versions --ignore-not-found=true
kubectl apply -f samples/app/meshservice.yaml
```

## 查看 xDS 推送结果

```bash
kubectl -n app logs -f deploy/nginx-consumer
```

日志里会看到类似输出：

```text
nginx.app.svc.cluster.local:80 v1=23 endpoints=1,v2=77 endpoints=1
```

应用容器会拿到这些关键变量：

- `GRPC_XDS_BOOTSTRAP`
- `XDS_ADDRESS`
- `CA_ADDRESS`

## 验证流量结果

```bash
kubectl -n app exec deploy/nginx-consumer -- dubbod xclient 100 | sort | uniq -c
```

`xclient` 现在是逐请求选路，不再先算好固定数量再批量发送。

## 在线更新验证

先在一个终端里持续发请求：

```bash
kubectl -n app exec deploy/nginx-consumer -- dubbod xclient --request-interval 200ms 200
```

再在另一个终端里修改 `MeshService` 权重。`nginx-consumer` 不需要重启，同一个 `xclient` 进程会继续使用同一条 xDS stream，后续请求会按新权重切换。

## 清理

```bash
kubectl -n app delete meshservice nginx-routing --ignore-not-found=true
kubectl -n app delete virtualservice nginx-weights --ignore-not-found=true
kubectl -n app delete destinationrule nginx-versions --ignore-not-found=true
kubectl delete -f samples/app/deployment.yaml --ignore-not-found=true
kubectl delete ns app
```
