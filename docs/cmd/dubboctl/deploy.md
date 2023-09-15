#Deploy

When you use dubboctl to install the required components, we will dynamically adjust the yaml file generated during the
deploy phase based on the components you installed. Specifically reflected in zookeeper, nacos and Prometheus. After you
install zookeeper, we will add information similar to this to the generated yaml file:

```yaml
         env:
           - name: zookeeper.address
             value: zookeeper.dubbo-system.svc
           - name: ZOOKEEPER_ADDRESS
             value: zookeeper.dubbo-system.svc
```

With these, you can use placeholders to read environment variables in your application to read the address of zookeeper
without having to fill it in manually, for example:

```yaml
Dubbo:
  application:
    logger: slf4j
    name: DemoApplication
  registry:
    address: zookeeper://${zookeeper.address:127.0.0.1}:2181
  protocol:
    name: tri
    port: 50051

```

The same goes for nacos.

For Prometheus, we will automatically generate a default list for Prometheus, which you can modify as needed.

## Advanced

If you installed related components in the dubbo-system namespace. Then the exact same components are installed in the
dev namespace, then we will give priority to using the addresses of related middleware in the dev namespace, and
everything will be in chronological order. Unless you specify it in an environment variable:

```sh
export DUBBO_DEPLOY_NS=dubbo-system
```