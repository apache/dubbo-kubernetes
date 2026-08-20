## moviereview example

moviepage / details / reviews(v1,v2,v3) / ratings 六个服务，应用代码本身是裸 HTTP、监听 9080，不依赖任何 Dubbo SDK。

这组清单不含任何网格专用字段，只作为普通 Kubernetes 应用运行。应用接入 Inherent
需要原生消费注入的 xDS bootstrap；仅给命名空间加注入标签不会把裸 HTTP 应用变成网格应用。

### 构建镜像

```bash
samples/moviereview/build.sh                                   # 构建到本地
HUB=kdubbo TAG=latest PUSH=true samples/moviereview/build.sh   # 构建并推送
PLATFORM=linux/amd64 samples/moviereview/build.sh              # 集群节点架构与本机不一致时
```

### 部署

```bash
kubectl create ns moviereview
kubectl label ns moviereview dubbo-injection=enabled   # 可选，纳入网格时才需要
kubectl apply -f samples/moviereview/deployment.yaml
kubectl -n moviereview rollout status deploy/moviepage
kubectl -n moviereview get svc frontend
```

访问 `http://<NodeIP>:30980`。
