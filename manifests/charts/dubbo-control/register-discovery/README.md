# Register-Discovery

![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)
![Helm: v3](https://img.shields.io/static/v1?label=Helm&message=v3&color=informational&logo=helm)

## Values

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