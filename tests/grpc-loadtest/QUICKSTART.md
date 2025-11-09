# Quick Start Guide

快速开始指南，帮助您快速运行 gRPC 负载测试。

## 前置要求

1. Go 1.24+
2. Protocol Buffers compiler (protoc)
3. Docker (可选，用于构建镜像)
4. Kubernetes 集群 (用于部署测试)

## 本地构建和测试

### 1. 生成 Proto 代码

```bash
cd tests/grpc-loadtest
make proto
```

或者手动运行：

```bash
cd proto
chmod +x gen.sh
./gen.sh
```

### 2. 下载依赖

```bash
go mod download
go mod tidy
```

### 3. 构建应用程序

```bash
make build
```

或者分别构建：

```bash
make server
make client
```

### 4. 运行服务器

在一个终端中启动服务器：

```bash
# 基准模式
./server -port=8080 -mode=baseline

# Envoy 模式
./server -port=8080 -mode=envoy

# xDS 无代理模式
./server -port=8080 -mode=xds
```

### 5. 运行客户端测试

在另一个终端中运行负载测试：

```bash
# 测试基准模式
./client -target=localhost:8080 -mode=baseline -qps=100 -duration=60s -connections=4

# 测试 Envoy 模式
./client -target=localhost:8080 -mode=envoy -qps=100 -duration=60s -connections=4

# 测试 xDS 无代理模式
./client -target=xds:///localhost:8080 -mode=xds -qps=100 -duration=60s -connections=4
```

## Kubernetes 部署

### 1. 构建 Docker 镜像

```bash
make docker
```

或者手动构建：

```bash
docker build -t loadtest-server:latest -f Dockerfile .
docker build -t loadtest-client:latest -f Dockerfile .
```

### 2. 推送到镜像仓库（如果需要）

```bash
docker tag loadtest-server:latest your-registry/loadtest-server:latest
docker push your-registry/loadtest-server:latest

docker tag loadtest-client:latest your-registry/loadtest-client:latest
docker push your-registry/loadtest-client:latest
```

### 3. 更新 Kubernetes 清单中的镜像名称

编辑 `k8s/*.yaml` 文件，将 `image: loadtest-server:latest` 替换为您的镜像仓库地址。

### 4. 部署服务器

```bash
kubectl apply -f k8s/baseline-server.yaml
kubectl apply -f k8s/envoy-server.yaml
kubectl apply -f k8s/xds-server.yaml
```

### 5. 运行自动化测试

```bash
./run-tests.sh
```

或者手动运行单个测试：

```bash
# 编辑 k8s/client-job.yaml 中的参数，然后：
kubectl apply -f k8s/client-job.yaml
kubectl logs -f job/loadtest-client-baseline -n grpc-loadtest
```

## 测试参数说明

### 客户端参数

- `-target`: 目标服务器地址
  - 基准/Envoy 模式: `localhost:8080` 或 `service.ns.svc.cluster.local:8080`
  - xDS 模式: `xds:///service.ns.svc.cluster.local:8080`
- `-mode`: 客户端模式 (`baseline`, `envoy`, `xds`)
- `-qps`: 每秒查询数 (默认: 10)
- `-duration`: 测试持续时间 (例如: `30s`, `5m`)
- `-connections`: 并发连接数 (默认: 1)
- `-payload`: 负载大小（字节）(默认: 0)
- `-timeout`: 请求超时时间 (默认: 30s)

### 服务器参数

- `-port`: gRPC 服务器端口 (默认: 8080)
- `-mode`: 服务器模式 (`baseline`, `envoy`, `xds`)

## 环境变量

### xDS 模式

对于 xDS 无代理模式，需要设置：

```bash
export GRPC_XDS_BOOTSTRAP=/etc/dubbo/proxy/grpc-bootstrap.json
```

在 Kubernetes 中，这个文件通常由 `dubbo-proxy` sidecar 生成。

## 故障排查

### Proto 代码生成失败

确保安装了 protoc 和相关的 Go 插件：

```bash
# 安装 protoc-gen-go
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

# 安装 protoc-gen-go-grpc
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### xDS 连接失败

1. 检查 `GRPC_XDS_BOOTSTRAP` 环境变量是否正确设置
2. 检查 bootstrap 文件是否存在且有效
3. 确保 Dubbo 控制平面正在运行
4. 检查 `dubbo-proxy` sidecar 是否正常运行

### Envoy 模式连接失败

1. 确保 Istio 已安装并正确配置
2. 检查 sidecar 注入是否成功：
   ```bash
   kubectl get pod -n grpc-loadtest -o jsonpath='{.items[*].spec.containers[*].name}'
   ```
3. 检查 Envoy sidecar 日志

## 性能对比

运行所有三种模式的测试后，比较以下指标：

1. **延迟百分位数**: P50, P90, P95, P99, P99.9
2. **吞吐量**: 实际达到的 QPS
3. **错误率**: 失败请求的百分比
4. **资源使用**: CPU 和内存消耗

## 下一步

- 查看 [README.md](README.md) 了解详细文档
- 调整测试参数以匹配您的环境
- 收集和分析性能指标

