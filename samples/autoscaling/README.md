## autoscaling example

用 KEDA 把队列消费者从 0 弹到 N，同时保持网格的 mTLS 和 xDS 不变。

KEDA 不是本项目的依赖，chart 不安装也不托管它。这里的 `ScaledObject` 是原生 KEDA 资源，
脱离网格照样有效。

### 前置

```bash
helm repo add kedacore https://kedacore.github.io/charts
helm install keda kedacore/keda -n keda --create-namespace
```

### 部署

```bash
kubectl create ns autoscaling
kubectl label ns autoscaling dubbo-injection=enabled
kubectl apply -f samples/autoscaling/kafka-consumer.yaml
kubectl apply -f samples/autoscaling/scaledobject.yaml

kubectl -n autoscaling get scaledobject order-consumer
kubectl -n autoscaling get deploy order-consumer -w
```

topic 空闲时副本数降到 0，有积压时自动拉起。

### 哪些工作负载可以缩到零

| 类型 | 策略 | 原因 |
| --- | --- | --- |
| 队列消费者、定时任务 | 0↔N | 没有入向调用方，副本消失不会让谁的请求落空 |
| 从网关进来的 Unary 服务 | 0↔N | 网关能扣住第一个请求等扩容，见 [`samples/activation`](../activation) |
| 只被其他服务调用的 Unary 服务 | 1↔N | 东西向没有网关，调用方直接读 EDS，缩到 0 后当场失败 |
| 启动慢的 Java 服务 | 1↔N + 预热 | 冷启动几分钟，扣住请求也等不起 |
| Streaming、长连接 | 不缩到 0 | 缩容会切断进行中的流，重放不了 |
| StatefulSet、强状态服务 | 不缩到 0 | 副本身份和存储不是可丢弃的 |

### 为什么东西向的服务仍然不能缩到零

出向是 proxyless 的：调用方进程内的 gRPC xDS client 直接拿 EDS 端点建连。副本归零时
dubbod 下发一个空的 CLA（`dubbod/discovery/pkg/xds/endpoints/endpoint_builder.go`），
调用方立即失败。服务之间的调用路径上没有任何组件能扣住这个请求去触发扩容 —— sidecar
网格里由 sidecar 承担的那个位置，在这里是空的。

南北向不同：请求进了网关就已经在网关自己的 task 里，可以等。这就是
[`samples/activation`](../activation) 做的事。

所以判断标准是**这个服务的调用方从哪来**，而不是它说什么协议。只被其他服务调用的，
用 `minReplicaCount: 1`。

### 副本数只能有一个归属

`ScaledObject` 一旦生效，`Deployment.spec.replicas` 就交给 KEDA 了。两边都写会让
KEDA 和 Deployment 控制器来回改同一个数字。`kafka-consumer.yaml` 里因此没有 `replicas` 字段。

同理，不要给受 KEDA 管的 Deployment 再挂一个 HPA。

### 缩容时的排空

被缩掉的 pod 会走注入的 dxplane 排空流程：先 5s 只失败 readiness（让 EndpointSlice 摘掉），
再关监听并等在途连接最多 25s。`terminationGracePeriodSeconds` 必须大于两者之和，
样例里设的是 40s。设成默认 30s 也够，但没有余量。

排空是否超预算，看 `dxplane_connections_force_closed_total`：非零说明 25s 不够，
调 `DUBBO_GRPC_INBOUND_TERMINATION_DRAIN_DURATION`。

### 冷启动的代价

每个新副本都要重新取证书、建 xDS 连接、等首次 CDS/EDS 收敛，这笔开销在 sidecar 方案里
由常驻 sidecar 摊销掉，proxyless 下每个副本重付。上生产前先量一下从 Pod Running 到
能收发流量的时间，再决定 `cooldownPeriod` 和 `stabilizationWindowSeconds`。

### 检查高可用配置

```bash
dubboctl analyze -n autoscaling
```

会报出单副本的控制面和网关、缺 PodDisruptionBudget、以及副本全落在同一个节点上的情况。
