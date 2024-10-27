## dubboctl dashboard zipkin

create PortForward between local address and target component zipkin pod. open browser by default

### Synopsis

create PortForward between local address and target component zipkin pod. open browser by default

Typical use cases are:

```sh

dubboctl dashboard zipkin

```

| parameter     | shorthand | describe                                                                                                         | Example                                                  | required |
|---------------|-----------|------------------------------------------------------------------------------------------------------------------|----------------------------------------------------------|----------|
| --port        | -p        | Set the port for local monitoring connections. If not set, the default is the same as the zipkin dashboard, 3000 | dubboctl  dashboard nacos -p 8848                        | No       |
| --host        | -h        | Set the local host to listen for connections. If not set, the default is 127.0.0.1                               | dubboctl dashboard nacos -h xxx.xxx.xxx.xxx              | No       |
| --openBrowser |           | Set whether to automatically open the browser and open the dashboard, the default is true                        | dubboctl dashboard nacos --openBrowser false             | No       |
| --namespace   | n         | Set the namespace where zipkin is located. If not set, the default is dubbo-system.                              | dubboctl dashboard nacos -n ns_user_specified            | No       |
| --kubeConfig  |           | The path to store kubeconfig                                                                                     | dubboctl dashboard nacos --kubeConfig path/to/kubeConfig | No       |
| --context     |           | Specify to use the context in kubeconfig                                                                         | dubboctl dashboard nacos --context contextVal            | No       |

### SEE ALSO

* [dubboctl dashboard](../dubboctl_dashboard.md) - Commands help user to open control plane components dashboards directly.