# 多集群主从跨网

## 结论

`chore_v47` 进入主从跨网方案：主集群运行 `dubbod`，远端集群通过 Secret 接入主控制面；跨集群流量不再直连远端 Pod IP，而是通过各集群的 east-west `dxgate` 转发。

## 前提

- 主集群 API Server 能访问远端 API Server。
- 两个集群共享同一信任根。
- 两个集群已安装 Gateway API CRD。
- 远端 webhook 和远端 `dxgate` 都能访问主集群控制面地址。
- 跨网模式允许 Pod CIDR 重叠，但每个集群都必须有可被其他集群访问的 east-west gateway 地址。

## 安装主控制面

主集群安装时同时打开远端访问和 east-west 配置：

```bash
dubboctl manifest generate \
  --set values.global.multicluster.remoteAccess.enabled=true \
  --set values.global.multicluster.remoteAccess.serviceType=NodePort \
  --set values.global.multicluster.remoteAccess.grpcPort=26010 \
  --set values.global.multicluster.remoteAccess.certificateHosts[0]=<主集群对外地址> \
  --set values.global.multicluster.eastWestGateway.enabled=true \
  --set values.global.multicluster.eastWestGateway.serviceType=NodePort \
  --set values.global.multicluster.eastWestGateway.nodePort=32443 \
  --set values.global.multicluster.eastWestGateway.gateways[0].clusterName=remote \
  --set values.global.multicluster.eastWestGateway.gateways[0].address=<远端 east-west 地址> \
  --set values.global.multicluster.eastWestGateway.gateways[0].port=15443 \
  --set values.global.multicluster.eastWestGateway.gateways[1].clusterName=Kubernetes \
  --set values.global.multicluster.eastWestGateway.gateways[1].address=<主集群 east-west 地址> \
  --set values.global.multicluster.eastWestGateway.gateways[1].port=15443 | kubectl --context cluster1 apply -f -
```

`DUBBO_EASTWEST_GATEWAYS` 会写入 `dubbod` Deployment。EDS 下发时，本集群 endpoint 保持 Pod IP，远端 cluster shard 会改写成对应 gateway 地址。

## 接入远端集群

主集群创建远端 Secret：

```bash
dubboctl multicluster create-remote-secret \
  --cluster-name remote \
  --kubeconfig ~/.kube/config2 \
  --context kubernetes-admin@kubernetes | kubectl --context cluster1 apply -f -
```

远端集群安装注入 webhook：

```bash
dubboctl multicluster generate-remote-manifest \
  --cluster-name remote \
  --webhook-url https://<主集群远端 webhook 地址>:<端口> \
  --xds-address <主集群远端 xDS/CA 地址>:<端口> \
  --ca-address <主集群远端 xDS/CA 地址>:<端口> \
  --ca-bundle-file ./ca-cert.pem | kubectl --context cluster2 apply -f -
```

远端集群创建 east-west Gateway；主控制面会通过远端 Secret 监听该 Gateway，并在远端生成 `dxgate`：

```bash
dubboctl multicluster generate-eastwest-gateway \
  --xds-address http://<主集群可访问 ADS 地址>:<grpc-xds NodePort 或 LB 端口> \
  --service-type NodePort \
  --node-port 32443 | kubectl --context cluster2 apply -f -
```

## 部署样例

```bash
for ctx in cluster1 cluster2; do
  kubectl --context "${ctx}" create ns app --dry-run=client -o yaml | kubectl --context "${ctx}" apply -f -
  kubectl --context "${ctx}" label namespace app dubbo-injection=enabled --overwrite
  kubectl --context "${ctx}" apply -f samples/app/deployment.yaml
done

kubectl --context cluster1 -n app scale deploy/nginx-v2 --replicas=0
kubectl --context cluster2 -n app scale deploy/nginx-v1 --replicas=0
kubectl --context cluster1 apply -f samples/app/meshservice.yaml
kubectl --context cluster1 apply -f samples/multicluster/eastwest-nginx-httproute.yaml
```

`HTTPRoute` 只需要放在主控制面的配置源里；远端 `dxgate` 连接主控制面 ADS 后会拿到同一份路由配置，并把流量转给远端本地 `nginx` Service。

## 验证

```bash
kubectl --context cluster2 -n app get pod -l app=nginx-consumer \
  -o jsonpath='{.items[0].spec.containers[0].env[?(@.name=="DUBBO_META_CLUSTER_ID")].value}{"\n"}'

kubectl --context cluster1 -n app exec deploy/nginx-consumer -- \
  dubbod xclient --print-route --expect v1=50,v2=50

kubectl --context cluster2 -n app exec deploy/nginx-consumer -- \
  dubbod xclient --print-route --expect v1=50,v2=50
```

跨网模式下，`--print-route` 里远端 endpoint 应显示 east-west gateway 地址，而不是远端 Pod IP。
