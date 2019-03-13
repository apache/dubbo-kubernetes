# dubbo-kubernetes
Dubbo integration with k8s


# 初步思考
kubernetes是天然可作为微服务的地址注册中心，类似于zookeeper， 阿里巴巴内部用到的VIPserver，Configserver。 具体来说，kubernetes中的Pod是对于应用的运行实例，Pod的被调度部署/启停都会调用API-Server的服务来保持其状态到ETCD；kubernetes中的service是对应微服务的概念，定义如下

A Kubernetes Service is an abstraction layer which defines a logical set of Pods and enables external traffic exposure, load balancing and service discovery for those Pods.

概括来说kubernetes service具有如下特点

每个Service都有一个唯一的名字，及对应IP。IP是kubernetes自动分配的，名字是开发者自己定义的。
Service的IP有几种表现形式，分别是ClusterIP，NodePort,LoadBalance,Ingress。 ClusterIP主要用于集群内通信；NodePort，Ingress，LoadBalance用于暴露服务给集群外的访问入口。

乍一看，kubernetes的service都是唯一的IP，在原有的Dubbo/HSF固定思维下：Dubbo/HSF的service是有整个服务集群的IP聚合而成，貌似是有本质区别的，细想下来差别不大，因为kubernetes下的唯一IP只是一个VIP，背后挂在了多个endpoint，那才是事实上的处理节点。

此处只讨论集群内的Dubbo服务在同一个kubernetes集群内访问；至于kubernetes外的consumer访问kubernetes内的provider，涉及到网络地址空间的问题，一般需要GateWay/loadbalance来做映射转换，不展开讨论。针对kubernetes内有两种方案可选：

- DNS： 默认kubernetes的service是靠DNS插件(最新版推荐是coreDNS)， Dubbo上有个 [proposal](https://github.com/apache/incubator-dubbo/issues/2043) 是关于这个的。我的理解是static resolution的机制是最简单最需要支持的一种service discovery机制，具体也可以参考Envoy在此的观点，由于HSF/Dubbo一直突出其软负载的地址发现能力，反而忽略Static的策略。同时蚂蚁的SOFA一直是支持此种策略，那一个SOFA工程的工程片段做一个解释。这样做有两个好处，1）当软负载中心crash不可用造成无法获取地址列表时，有一定的机制Failover到此策略来处理一定的请求。 2）在LDC/单元化下，蚂蚁的负载中心集群是机房/区域内收敛部署的，首先保证软负载中心的LDC化了进而稳定可控，当单元需要请求中心时，此VIP的地址发现就排上用场了。 

- API：DNS是依靠DNS插件进行的，相当于额外的运维开销，所以考虑直接通过kubernetes的client来获取endpoint。事实上，通过访问kubernetes的API server接口是可以直接获取某个servie背后的endpoint列表，同时可以监听其地址列表的变化。从而实现Dubbo/HSF所推荐的软负载发现策略。具体可以参考代码：

以上两种思路都需要考虑以下两点
- kubernetes和Dubbo对于service的名字是映射一致的。Dubbo的服务是由serviename，group，version三个来确定其唯一性，而且servicename一般其服务接口的包名称，比较长。需要映射kubernetes的servie名与dubbo的服务名。要么是像SOFA那样增加一个属性来进行定义，这个是改造大点，但最合理；要么是通过固定规则来引用部署的环境变量，可用于快速验证。
- 端口问题。默认Pod与Pod的网络互通算是解决了。需要验证。
