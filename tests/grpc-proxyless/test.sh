#!/bin/bash
set -e

NAMESPACE="grpc-proxyless"

echo "=== 测试 gRPC Proxyless 死循环修复 ==="
echo ""

# 1. 检查控制平面日志（确认死循环已解决）
echo "1. 检查控制平面日志（观察 10 秒，确认不再出现死循环）..."
echo "   运行: kubectl logs -f <dubbod-pod-name> -n <namespace> | grep -E 'LDS.*PUSH|resources:'"
echo "   期望: 不再出现 resources:1 和 resources:13 的快速交替"
echo ""

# 2. 检查 xDS proxy 日志
echo "2. 检查 xDS proxy 日志（确认不再频繁转发）..."
CONSUMER_POD=$(kubectl get pods -n $NAMESPACE -l app=consumer -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
if [ -n "$CONSUMER_POD" ]; then
    echo "   检查 pod: $CONSUMER_POD"
    echo "   运行: kubectl logs $CONSUMER_POD -c dubbo-proxy -n $NAMESPACE | grep -E 'forwarding request.*LDS' | wc -l"
    echo "   期望: 数量很少，不再频繁转发"
else
    echo "   未找到 consumer pod"
fi
echo ""

# 3. 获取服务信息
echo "3. 获取服务信息..."
kubectl get svc -n $NAMESPACE
echo ""

# 4. 使用 grpcurl 测试
echo "4. 使用 grpcurl 测试服务功能..."
echo ""

# 测试 ForwardEcho（通过 producer）
PRODUCER_POD=$(kubectl get pods -n $NAMESPACE -l app=producer -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
if [ -n "$PRODUCER_POD" ]; then
    echo "   测试 ForwardEcho（通过 producer pod）:"
    echo "   kubectl port-forward -n $NAMESPACE $PRODUCER_POD 17171:17171 &"
    echo "   grpcurl -plaintext localhost:17171 echo.EchoTestService/ForwardEcho -d '{\"url\": \"xds:///consumer.grpc-proxyless.svc.cluster.local:7070\", \"count\": 1, \"headers\": {}, \"timeout\": 5}'"
    echo ""
fi

# 测试 Echo（直接访问 consumer）
CONSUMER_POD=$(kubectl get pods -n $NAMESPACE -l app=consumer -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
if [ -n "$CONSUMER_POD" ]; then
    echo "   测试 Echo（直接访问 consumer pod）:"
    echo "   kubectl port-forward -n $NAMESPACE $CONSUMER_POD 17070:17070 &"
    echo "   grpcurl -plaintext localhost:17070 echo.EchoService/Echo -d '{\"message\": \"test\"}'"
    echo ""
fi

echo "=== 测试完成 ==="


