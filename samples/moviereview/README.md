## moviereview example

moviepage / details / reviews(v1,v2,v3) / ratings 六个服务，应用代码本身是裸 HTTP、监听 9080，不依赖任何 Dubbo SDK。

**这个示例必须在开启注入的 namespace 里运行。** 注入后 CNI 会拦截 pod 入站流量，
9080 对 kubelet 不可达，健康检查只能走 sidecar 的 15080 端口 —— `deployment.yaml`
里的探针端口就是按这个前提写的。不打注入标签，Pod 会因为探针失败而反复重启。

### 构建镜像

```bash
samples/moviereview/build.sh                                   # 构建到本地
HUB=kdubbo TAG=latest PUSH=true samples/moviereview/build.sh   # 构建并推送
PLATFORM=linux/amd64 samples/moviereview/build.sh              # 集群节点架构与本机不一致时
```

### 部署

```bash
kubectl create ns moviereview
kubectl label ns moviereview dubbo-injection=enabled   # 必需
kubectl apply -f samples/moviereview/deployment.yaml
kubectl -n moviereview rollout status deploy/moviepage
kubectl -n moviereview get svc frontend
```

访问 `http://<NodeIP>:30980`。
