# Dubbo 可观测性 Addons

这个目录提供开箱即用的 Prometheus 和 Grafana 部署，用于观察 `dubbo-system/dubbod` 的日志信号、监控指标和控制面追踪耗时。

## 安装

```bash
kubectl apply -f samples/addons
kubectl -n dubbo-system rollout status deploy/prometheus --timeout=180s
kubectl -n dubbo-system rollout status deploy/grafana --timeout=180s
```

## 打开 Grafana

```bash
kubectl -n dubbo-system port-forward svc/grafana 3000:3000
```

浏览器打开 `http://127.0.0.1:3000`，进入 `Dubbo / Dubbo Control Plane Observability` 仪表盘。

## 验证 Prometheus

```bash
kubectl -n dubbo-system port-forward svc/prometheus 9090:9090
curl 'http://127.0.0.1:9090/api/v1/query?query=dubbod_uptime_seconds'
```

## 覆盖内容

- 日志：`dubbod_log_messages_total` 按 `level` 和 `scope` 统计日志速率，原始日志仍通过 `kubectl -n dubbo-system logs deploy/dubbod` 查看。
- 监控：Prometheus 抓取 `dubbod.dubbo-system.svc:8080/metrics`，Grafana 内置控制面总览面板。
- 追踪：通过 `dubbod_xds_push_time`、`dubbod_xds_send_time`、`dubbod_proxy_queue_time` 等直方图展示 xDS 推送链路耗时。

## 清理

```bash
kubectl delete -f samples/addons --ignore-not-found=true
```
