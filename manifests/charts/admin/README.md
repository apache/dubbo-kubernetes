# Admin         

![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)
![Helm: v3](https://img.shields.io/static/v1?label=Helm&message=v3&color=informational&logo=helm)

## Values

### Admin

| Key                                              | Description                                                                                | Default                                   |
|--------------------------------------------------|--------------------------------------------------------------------------------------------|-------------------------------------------|
| `deployType`                                     | Define the deployment mode for Admin.                                                      | `Deployment`                              |
| `auth.enabled`                                   | Auth Status for Admin Control Plane.                                                       | `true`                                    |
| `auth.authorization.action`                      | Define the Authorization Action for Admin Control Plane.                                   | `DENY`                                    |
| `auth.authorization.matchType`                   | Define the Authorization MatchType for Admin Control Plane.                                | `anyMatch`                                |
| `auth.authorization.samples`                     | Define the rule sampling rate for Authorization the Admin Control Plane.                   | `0`                                       |
| `auth.authentication.action`                     | Define the Authentication Action for Admin Control Plane.                                  | `STRICT`                                  |
| `auth.authentication.port`                       | Define the port number for applying the Authentication Policy for the Admin Control Plane. | `8080`                                    |
| `traffic.enabled`                                | Traffic Status for Admin Control Plane.                                                    | `true`                                    |
| `traffic.conditionRoute.scope`                   | Supports service and application scope rules.                                              | `service`                                 |
| `traffic.conditionRoute.enabled`                 | Whether enable this rule or not, set enabled:false to disable this rule                    | `true`                                    |
| `traffic.conditionRoute.force`                   | The behaviour when the instance subset is empty after routing.                             | `true`                                    |
| `traffic.conditionRoute.runtime`                 | Whether run routing rule for every rpc invocation or use routing cache if available.       | `true`                                    |
| `traffic.conditionRoute.priority`                | Specify the specific priority for traffic.                                                 | `100`                                     |
| `traffic.conditionRoute.configVersion`           | The version of the condition rule definition, currently available version is v3.0.         | `v3.0`                                    |
| `traffic.conditionRoute.key`                     | The identifier of the target service or application that this rule is about to apply to.   | `org.apache.dubbo.samples.CommentService` |
| `traffic.conditionRoute.conditions`              | The condition routing rule definition of this configuration. Check Condition for details.  | `method=getComment => region=Hangzhou`    |
| `traffic.dynamicConfig.scope`                    | Supports service and application scope rules.                                              | `service`                                 |
| `traffic.dynamicConfig.configVersion`            | The version of the tag rule definition, currently available version is v3.0.               | `v3.0`                                    |
| `traffic.dynamicConfig.key`                      | The identifier of the target service or application that this rule is about to apply to.   | `org.apache.dubbo.samples.UserService`    |
| `traffic.dynamicConfig.side`                     | Especially useful when scope:service is set.                                               | `consumer`                                |
| `traffic.dynamicConfig.exact`                    | The application matching condition for this config rule to take effect.                    | `shop-frontend`                           |
| `traffic.tagRoute.name`                          | The name of the tag used to match the dubbo tag value in the request context.              | `gray`                                    |
| `traffic.tagRoute.enabled`                       | Whether enable this rule or not, set enabled:false to disable this rule.                   | `false`                                   |
| `traffic.tagRoute.force`                         | The behaviour when the instance subset is empty after routing.                             | `true`                                    |
| `traffic.tagRoute.configVersion`                 | The version of the tag rule definition, currently available version is v3.0.               | `v3.0`                                    |
| `traffic.tagRoute.priority`                      | Specify the specific priority for traffic.                                                 | `99`                                      |
| `traffic.tagRoute.key`                           | The identifier of the target application that this rule is about to control.               | `details`                                 |

### ZooKeeper

| Key                                                           | Description                                                                                                                 | Default                       |
|---------------------------------------------------------------|-----------------------------------------------------------------------------------------------------------------------------|-------------------------------|
| `zookeeper.enabled`                                           | ZooKeeper Status for Admin.                                                                                                 | `true`                        |
| `zookeeper.dataLogDir`                                        | Dedicated data log directory.                                                                                               | `""`                          |
| `zookeeper.tickTime`                                          | Basic time unit (in milliseconds) used by ZooKeeper for heartbeats.                                                         | `2000`                        |
| `zookeeper.initLimit`                                         | ZooKeeper uses to limit the length of time the ZooKeeper servers in quorum have to connect to a leader.                     | `10`                          |
| `zookeeper.syncLimit`                                         | How far out of date a server can be from a leader.                                                                          | `5`                           |
| `zookeeper.preAllocSize`                                      | Block size for transaction log file.                                                                                        | `65536`                       |
| `zookeeper.snapCount`                                         | The number of transactions recorded in the transaction log before a snapshot can be taken (and the transaction log rolled). | `100000`                      |
| `zookeeper.fourlwCommandsWhitelist`                           | A list of comma separated Four Letter Words commands that can be executed.                                                  | `srvr, mntr, ruok`            |
| `zookeeper.listenOnAllIPs`                                    | Allow ZooKeeper to listen for connections from its peers on all available IP addresses.                                     | `false`                       |
| `zookeeper.autopurge.snapRetainCount`                         | The most recent snapshots amount (and corresponding transaction logs) to retain.                                            | `3`                           |
| `zookeeper.autopurge.purgeInterval`                           | The time interval (in hours) for which the purge task has to be triggered.                                                  | `0`                           |
| `zookeeper.maxClientCnxns`                                    | Limits the number of concurrent connections that a single client may make to a single member of the ZooKeeper ensemble.     | `60`                          |
| `zookeeper.maxSessionTimeout`                                 | Maximum session timeout (in milliseconds) that the server will allow the client to negotiate.                               | `40000`                       |
| `zookeeper.heapSize`                                          | Size (in MB) for the Java Heap options (Xmx and Xms).                                                                       | `1024`                        |
| `zookeeper.logLevel`                                          | Log level for the ZooKeeper server. ERROR by default.                                                                       | `ERROR`                       |
| `zookeeper.auth.client.enabled`                               | Enable ZooKeeper client-server authentication. It uses SASL/Digest-MD5.                                                     | `false`                       |
| `zookeeper.auth.client.clientUser`                            | User that will use ZooKeeper clients to auth.                                                                               | `""`                          |
| `zookeeper.auth.client.clientPassword`                        | Password that will use ZooKeeper clients to auth.                                                                           | `""`                          |
| `zookeeper.auth.client.serverUsers`                           | Comma, semicolon or whitespace separated list of user to be created.                                                        | `""`                          |
| `zookeeper.auth.client.serverPasswords`                       | Comma, semicolon or whitespace separated list of passwords to assign to users when created.                                 | `""`                          |
| `zookeeper.auth.client.existingSecret`                        | Use existing secret (ignores previous passwords).                                                                           | `""`                          |
| `zookeeper.auth.quorum.enabled`                               | Enable ZooKeeper server-server authentication. It uses SASL/Digest-MD5.                                                     | `false`                       |
| `zookeeper.auth.quorum.learnerUser`                           | User that the ZooKeeper quorumLearner will use to authenticate to quorumServers.                                            | `""`                          |
| `zookeeper.auth.quorum.learnerPassword`                       | Password that the ZooKeeper quorumLearner will use to authenticate to quorumServers.                                        | `""`                          |
| `zookeeper.auth.quorum.serverUsers`                           | Comma, semicolon or whitespace separated list of users for the quorumServers.                                               | `""`                          |
| `zookeeper.auth.quorum.serverPasswords`                       | Comma, semicolon or whitespace separated list of passwords to assign to users when created.                                 | `""`                          |
| `zookeeper.auth.quorum.existingSecret`                        | Use existing secret (ignores previous passwords).                                                                           | `""`                          |

### Nacos

| Key                                                       | Description                                                                                                 | Default                                                                                         |
|-----------------------------------------------------------|-------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------|
| `nacos.enabled`                                           | Nacos Status for Admin.                                                                                     | `false`                                                                                         |
| `nacos.mode`                                              | Run Mode standalone or cluster.                                                                             | `standalone`                                                                                    |
| `nacos.clusterDomain`                                     | Cluster Domain Suffix for Nacos.                                                                            | `cluster.local`                                                                                 |
| `nacos.plugin.enabled`                                    | Plugin Status for Nacos.                                                                                    | `true`                                                                                          |
| `nacos.plugin.image.registry`                             | Plugin Image Name for Nacos.                                                                                | `nacos/nacos-peer-finder-plugin`                                                                |
| `nacos.plugin.image.tag`                                  | Plugin Version Tag for Nacos.                                                                               | `1.1`                                                                                           |
| `nacos.plugin.image.pullPolicy`                           | Plugin Pull Policy for Nacos.                                                                               | `IfNotPresent`                                                                                  |
| `nacos.serverPort`                                        | Define the service port for the Nacos.                                                                      | `8848`                                                                                          |
| `nacos.preferhostmode`                                    | Enable Nacos cluster node domain name support                                                               | `~`                                                                                             |
| `nacos.storage.type`                                      | Nacos data storage method `mysql` or `embedded`. The `embedded` supports either standalone or cluster mode. | `embedded`                                                                                      |
| `nacos.storage.db.host`                                   | Specify the database host for Nacos storing configuration data.                                             | `localhost`                                                                                     |
| `nacos.storage.db.name`                                   | Specify the database name for Nacos storing configuration data.                                             | `nacos`                                                                                         |
| `nacos.storage.db.port`                                   | Specify the database port for Nacos storing configuration data.                                             | `3306`                                                                                          |
| `nacos.storage.db.username`                               | Specify the database username for Nacos storing configuration data.                                         | `mysql`                                                                                         |
| `nacos.storage.db.password`                               | Specify the database password for Nacos storing configuration data.                                         | `passw0rd`                                                                                      |
| `nacos.storage.db.param`                                  | Specify the database url parameter for Nacos storing configuration data.                                    | `characterEncoding=utf8&connectTimeout=1000&socketTimeout=3000&autoReconnect=true&useSSL=false` |


