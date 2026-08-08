## activation example

把一个 HTTP/gRPC 服务缩到零，第一个请求到达时再拉起来。请求被网关扣住等扩容，不是失败重试。

和 [`samples/autoscaling`](../autoscaling) 的区别：那边是队列消费者，没有入向调用方，副本消失不会让谁的请求落空；这边是有人调用的服务，缩到零之后必须有东西在请求路径上接住第一个请求。

### 链路

```
请求 -> dxgate 扣住请求
          |  上报 pending 数（gRPC 流）
          v
       dubbod  <-- KEDA 轮询 external scaler
          |
          v
       ScaledObject -> Deployment 0 -> 1
          |
          v
       EDS 下发新端点 -> dxgate 放行被扣住的请求
```

南北向请求直接经过 dxgate。东西向请求原本由 proxyless 调用方直接读 EDS；冷服务的 EDS 现在会临时改写为专用 Activator `dxgate-gateway`，所以同一个扣流和上报机制也能接住服务间调用。后端就绪后，EDS 恢复真实端点，新请求直接访问后端。

网关只负责等和上报，副本数始终由 KEDA 写。这条边界是有意的：网关重启不会把某个工作负载留在没人要求过的副本数上。

### 前置

```bash
helm repo add kedacore https://kedacore.github.io/charts
helm install keda kedacore/keda -n keda --create-namespace
```

需要一个名为 `dxgate-gateway` 的 Gateway 作为命名空间内的专用 Activator。`dubbod` 会为它拉起 dxgate，并注入 `DXGATE_ACTIVATION_CONTROL_PLANE`。其他 Gateway 使用各自派生的 Deployment/Service 名称，不会覆盖或清理 Activator 资源。

### 部署

```bash
kubectl create ns activation
kubectl label ns activation dubbo-injection=enabled
kubectl apply -f samples/activation/payment.yaml
kubectl apply -f samples/activation/activation-policy.yaml
kubectl apply -f samples/activation/scaledobject.yaml
```

### 验证

等它自己缩到零（`cooldownPeriod: 300`，五分钟），或者直接改 `ScaledObject` 缩短：

```bash
kubectl -n activation get deploy payment -w
# READY 0/0
```

然后打一个请求：

```bash
time curl -s http://$GATEWAY/payment/healthz
```

请求会挂住几秒——那是冷启动——然后正常返回，不是 503。同一时刻看网关：

```bash
kubectl -n activation port-forward deploy/dxgate-gateway 15021:26021
curl -s localhost:15021/metrics | grep dxgate_activation_requests_held
# dxgate_activation_requests_held 1
```

这个指标和 `dxgate_requests_in_flight` 是分开的，因为它们要区别对待：前者是在等扩容，后者是在等上游。混在一起会让一次冷启动看起来像网关变慢了。

策略状态里能看到控制面这一侧是否就绪：

```bash
kubectl -n activation get serviceactivationpolicy payment -o jsonpath='{.status.conditions}' | jq
```

四个 condition：`Accepted` 策略本身合法，`Eligible` 目标可被激活，`ScalerReady` 引用的 KEDA `ScaledObject` 已 Ready，`ActivatorReady` 同命名空间至少一个 Dubbo Gateway 已 Programmed。后两项读取 Kubernetes 共享状态，HA 副本不会因各自持有不同连接而互相覆盖。

### 三个组件各自负责什么

| 组件 | 负责 | 不负责 |
| --- | --- | --- |
| `ServiceActivationPolicy` | 声明哪个 Service 可以被激活、扣多久、扣多少 | 不写副本数 |
| `ScaledObject` | 副本数的唯一归属 | 不知道请求被扣住这回事 |
| dxgate | 扣住请求、上报 pending、端点出现后放行 | 不扩容任何东西 |

少任何一个都不成立。只有策略没有 `ScaledObject`，请求会被扣满 `requestTimeout` 然后失败；只有 `ScaledObject` 没有策略，控制面不会为这个目标发布 scaler 指标，KEDA 拿不到 pending 数，服务永远停在零。

### 哪些请求不能被扣住

`protocols` 里只列了 `HTTP` 和 `GRPC_UNARY`。流式请求不在其中：一个流没法在目标起来之后重放，扣住它换到的冷启动，代价是这个流已经断了。长连接同理。

对这类服务用 `minReplicaCount: 1`。

### 第一个请求要等多久

四段相加：

1. KEDA 轮询间隔——样例里 `pollingInterval: 1`，最多 1s
2. HPA 把副本从 0 改到 1
3. Pod 调度 + 拉镜像 + 进程启动
4. proxyless 冷启动：取证书、建 xDS 连接、等首次 CDS/EDS 收敛

第 4 段是 sidecar 方案里由常驻 sidecar 摊销掉、而 proxyless 下每个副本都要重付的部分。上生产前量一下从 Pod Running 到能收发流量的时间，再定 `requestTimeout`——它必须大于这个总和，否则请求总在服务刚要就绪时被判超时。

`requestTimeout` 同时要小于调用方自己的超时。调用方已经放弃的请求，在网关这边多扣一会儿没有任何意义。

### 扣住的请求占用什么

每个被扣住的请求占一个 task 和一条连接。所以有两层上限：

- `maxPendingRequests`（策略级，样例 100）——单个冷目标能占掉网关多少
- `DXGATE_ACTIVATION_MAX_PENDING_REQUESTS`（网关级，默认 1024）——所有目标合计

超过上限的请求直接失败，不进等待队列，也不会出现在上报里。这是有意的：让一个起不来的目标拖垮整个网关，比这些请求早点失败要糟得多。

### 缩容时的排空

和 `samples/autoscaling` 一样，被缩掉的 pod 走 dxplane 的两阶段排空：先 5s 只失败 readiness 让 EndpointSlice 摘掉，再关监听等在途连接最多 25s。`terminationGracePeriodSeconds` 必须大于两者之和，`payment.yaml` 里是 40s。

`dxplane_connections_force_closed_total` 非零说明 25s 不够，调 `DUBBO_GRPC_INBOUND_TERMINATION_DRAIN_DURATION`。

### 为什么网关上报和 KEDA 查询使用不同地址

```yaml
scalerAddress: dubbod-activation.dubbo-system.svc.cluster.local:26030
```

ScaledObject 使用负载均衡的 `dubbod-activation`，无论 KEDA 落到哪个控制面副本都能拿到相同 pending。网关上报使用 headless 的 `dubbod-activation-replicas`，向解析出的**每一个**地址各上报一份，保证每个控制面副本都有相同数据。

同理，网关必须注入 `POD_NAME`：控制面按 reporter 身份聚合，两个网关副本共用一个身份会互相覆盖。这个变量由 `kube-gateway.yaml` 自动注入。

### 东西向和 mTLS

带 `ServiceActivationPolicy` 的冷服务不会收到空 EDS。`dubbod` 把端点临时改成同命名空间 `dxgate-gateway` 的地址；Activator RDS 再按原始 Host 路由到真实服务。扩容完成后只切 EDS，不切 CDS。

`backendServiceAccounts` 是生产必填项。CDS 的 `MatchSubjectAltNames` 始终包含后端身份和 Activator 身份，冷/热切换期间 SAN 集合保持不变，避免证书校验窗口。不要为了省配置使用通配 SAN。

### 生产边界

- 只支持 HTTP 和 unary gRPC；流式 RPC、长连接、启动时间超过调用方 deadline 的服务保持 `minReplicaCount: 1`。
- Activator 和 dubbod 都至少两个副本，并配置 PodDisruptionBudget。控制面 pending 是内存状态；全部控制面同时重启时，在网关下一次上报前 KEDA 暂时读到 0。
- `maxPendingRequests` 和网关全局 backlog 都要压测。满载时按 `failurePolicy` 快速失败，不承诺无限排队。
- 监控 `dxgate_activation_requests_held`、请求 4xx/5xx、KEDA ScaledObject/HPA 条件、策略的 `ScalerReady`/`ActivatorReady`。告警必须覆盖“pending 持续上升但副本仍为 0”。
- 升级先保持目标至少一个副本，升级 CRD/base、dubbod、dxgate 后确认两种 SAN 和 Activator RDS 已下发，再恢复 `minReplicaCount: 0`。
