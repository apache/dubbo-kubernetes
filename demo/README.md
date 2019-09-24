### 基本点
- provider只要把端口打开就行，不用自己注册。是通过pod注册到k8s的。
- consumer是需要通过k8s的fabric client来获取对应的服务地址。

### 问答

------------
 - A：我理解的是一个Pod一个Provider或者Consumer进程，剩下的那个k8s init-container 不会帮业务进程做任何事情的。那Provider怎么把自己注册到k8s中？
 - B：好问题。是k8s的 service.yaml 中发布的。用k8s的 方式定义出来的。为了复用k8s的service能力，就需要用k8s的方式发布服务。
 - A: provider问题解了， consumer订阅的时候就需要知道 k8s的API server地址。之前的话，不管是zk还是其他k/v介质，都是dubbo进程启动之后自己主动写进去注册中心的，所以我一直以为provider和consumer都是自个儿把自己的meta数据写到kubernetes。原来不是。。


------------
- 问：需要手工写yaml然后kubectl apply -f 么？
- 答：目前是的， 当然你可以写个maven插件什么的，通过扫描dubbo的配置来自动生成。我感觉是需要开发一个额外的Operator。

------------
- 问：还有个问题是 k8s发布的服务名与dubbo的服务名的转换问题。
- 答：这个转换映射目前是通过 -D参数配置的方式， 也是比较初级。假设dubbo服务名是 com.abc.service.XYZ, 那需要启动时配置 -Dcom.abc.service.XYZ_target=XYZ.service.com 之类的。k8s的服务命名方式一般是正序， 而dubbo是包名。

------------
- 问：目前有一个规范么？不然dubbo-go这边和dubbo如果不使用同一种方式，出来的 Provider/Consumer 无法正常工作。
- 答：规范需要大家一起来完善。