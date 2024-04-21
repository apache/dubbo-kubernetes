# Admin         

![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)
![Helm: v3](https://img.shields.io/static/v1?label=Helm&message=v3&color=informational&logo=helm)

## Values

### Admin

| Key                                              | Description                                                                                | Default                                   |
|--------------------------------------------------|--------------------------------------------------------------------------------------------|-------------------------------------------|
| `deployType`                                     | Define the deployment mode for Admin.                                                      | `Deployment`                              |
| `namespaceOverride`                              | NameSpace Override for Admin.                                                              | `~`                                       |
| `labels`                                         | Labels for Admin.                                                                          | `~`                                       |
| `annotations`                                    | Annotations for Admin.                                                                     | `~`                                       |
| `nodeSelector`                                   | Node Scheduling for Admin.                                                                 | `~`                                       |
| `imagePullSecrets`                               | Image Pull Credentials for Admin.                                                          | `~`                                       |
| `clusterDomain`                                  | Cluster Domain Suffix for Admin.                                                           | `cluster.local`                           |
| `replicas`                                       | Replica Deployment Count for Admin.                                                        | `1`                                       |
| `image.registry`                                 | Image Name for Admin.                                                                      | `docker.io/apache/dubbo-admin`            |
| `image.tag`                                      | Version Tag for Admin.                                                                     | `latest`                                  |
| `image.pullPolicy`                               | Pull Policy for Admin.                                                                     | `IfNotPresent`                            |
| `rbac.enabled`                                   | Role-Based Access Control Status for Admin.                                                | `true`                                    |
| `rbac.labels`                                    | Role-Based Access Control Labels Definition for Admin.                                     | `~`                                       |
| `rbac.annotations`                               | Role-Based Access Control Annotations Definition for Admin.                                | `~`                                       |
| `serviceAccount.enabled`                         | Service Accounts Status for Admin.                                                         | `true`                                    |
| `serviceAccount.labels`                          | Service Accounts Labels Definition for Admin.                                              | `~`                                       |
| `serviceAccount.annotations`                     | Service Accounts Annotations Definition for Admin.                                         | `~`                                       |
| `volumeMounts`                                   | Internal Mount Directory for Admin.                                                        | `~`                                       |
| `volumes`                                        | External Mount Directory for Admin.                                                        | `~`                                       |
| `configMap`                                      | ConfigMap Mount Configuration for Admin.                                                   | `~`                                       |
| `secret`                                         | Secret Mount Configuration for Admin.                                                      | `~`                                       |
| `strategy.type`                                  | Policy Type for Admin.                                                                     | `RollingUpdate`                           |
| `strategy.rollingUpdate.maxSurge`                | Admin Update Strategy Expected Replicas.                                                   | `%`                                       |
| `strategy.rollingUpdate.maxUnavailable`          | Admin Update Strategy Maximum Unavailable Replicas.                                        | `1`                                       |
| `updateStrategy.type`                            | Update Policy Type for Admin.                                                              | `RollingUpdate`                           |
| `updateStrategy.rollingUpdate`                   | Rolling Update Strategy for Admin.                                                         | `~`                                       |
| `minReadySeconds`                                | Seconds to Wait Before Admin Readiness.                                                    | `0`                                       |
| `revisionHistoryLimit`                           | Number of Revision Versions Saved in History for Admin.                                    | `10`                                      |
| `terminationGracePeriodSeconds`                  | Graceful Termination Seconds for Admin Hooks.                                              | `30`                                      |
| `startupProbe.initialDelaySeconds`               | Initialization Wait Time in Seconds After Admin Container Starts.                          | `60`                                      |
| `startupProbe.timeoutSeconds`                    | Response Timeout Duration in Seconds After Admin Container Starts.                         | `30`                                      |
| `startupProbe.periodSeconds`                     | The Admin container periodically checks availability.                                      | `10`                                      |
| `startupProbe.successThreshold`                  | The success threshold for the Admin container.                                             | `1`                                       |
| `startupProbe.httpGet.path`                      | Checking the container with an HTTP GET request to a target path.                          | `/`                                       |
| `startupProbe.httpGet.port`                      | Checking the container with an HTTP GET request to a target port.                          | `8080`                                    |
| `readinessProbe.initialDelaySeconds`             | Initialization Wait Time in Seconds After Admin Container Starts.                          | `60`                                      |
| `readinessProbe.timeoutSeconds`                  | Response Timeout Duration in Seconds After Admin Container Starts.                         | `30`                                      |
| `readinessProbe.periodSeconds`                   | The Admin container periodically checks availability.                                      | `10`                                      |
| `readinessProbe.successThreshold`                | The success threshold for the Admin container.                                             | `1`                                       |
| `readinessProbe.httpGet.path`                    | Checking the container with an HTTP GET request to a target path.                          | `/`                                       |
| `readinessProbe.httpGet.port`                    | Checking the container with an HTTP GET request to a target port.                          | `8080`                                    |
| `livenessProbe.initialDelaySeconds`              | Initialization Wait Time in Seconds After Admin Container Starts.                          | `60`                                      |
| `livenessProbe.timeoutSeconds`                   | Response Timeout Duration in Seconds After Admin Container Starts.                         | `30`                                      |
| `livenessProbe.periodSeconds`                    | The Admin container periodically checks availability.                                      | `10`                                      |
| `livenessProbe.successThreshold`                 | The success threshold for the Admin container.                                             | `1`                                       |
| `livenessProbe.httpGet.path`                     | Checking the container with an HTTP GET request to a target path.                          | `/`                                       |
| `livenessProbe.httpGet.port`                     | Checking the container with an HTTP GET request to a target port.                          | `8080`                                    |
| `lifecycleHooks`                                 | Graceful hooks for the Admin.                                                              | `~`                                       |
| `service.enabled`                                | Service Status for Admin.                                                                  | `true`                                    |
| `service.labels`                                 | Service Label Definition for Admin.                                                        | `~`                                       |
| `service.annotations`                            | Service Annotations Definition for Admin.                                                  | `~`                                       |
| `service.type`                                   | Define the service type for the Admin.                                                     | `ClusterIP`                               |
| `service.clusterIP`                              | Define the service cluster IP for the Admin.                                               | `~`                                       |
| `service.externalIPs`                            | Define the service external IP for the Admin.                                              | `~`                                       |
| `service.loadBalancerIP`                         | Define the service loadBalancer IP for the Admin.                                          | `~`                                       |
| `service.loadBalancerSourceRanges`               | Define the service loadBalancer Source Ranges for the Admin.                               | `~`                                       |
| `service.loadBalancerClass`                      | Define the service loadBalancer Class for the Admin.                                       | `~`                                       |
| `service.sessionAffinity`                        | Define the session affinity strategy for the Admin service.                                | `None`                                    |
| `service.publishNotReadyAddresses`               | Define the publication of not-ready Admin service addresses to other components.           | `true`                                    |
| `service.protocol`                               | Service Protocol Definition for Admin.                                                     | `TCP`                                     |
| `resources.limits.cpu`                           | Maximum Limit on CPU Resources for Admin.                                                  | `128`                                     |
| `resources.limits.memory`                        | Maximum Limit on Memory Resources for Admin.                                               | `128`                                     |
| `resources.requests.cpu`                         | Maximum Request on CPU Resources for Admin.                                                | `128`                                     |
| `resources.requests.memory`                      | Maximum Request on Memory Resources for Admin.                                             | `128`                                     |
| `tolerations`                                    | toleration's Definition for Admin.                                                         | `~`                                       |
| `persistence.enabled`                            | Persistence Status for Admin.                                                              | `false`                                   | 
| `persistence.labels`                             | Persistence Labels Definition for Admin.                                                   | `~`                                       |
| `persistence.annotations`                        | Persistence Annotations Definition for Admin.                                              | `~`                                       |
| `persistence.claimName`                          | Persistence claim name Definition for Admin.                                               | `""`                                      | 
| `persistence.storageclass`                       | Persistence storage class Definition for Admin.                                            | `""`                                      | 
| `persistence.size`                               | Persistence size Definition for Admin.                                                     | `1Gi`                                     |
| `persistence.accessModes`                        | Persistence accessModes Definition for Admin.                                              | `ReadWriteOnce`                           | 
| `securityContext.runAsNonRoot`                   | Whether the security context for Admin runs as a non-privileged user.                      | `false`                                   |
| `securityContext.runAsUser`                      | Non-privileged user identifier for Admin security context.                                 | `1000`                                    |
| `securityContext.runAsGroup`                     | Non-privileged group identity identifier for Admin security context.                       | `1000`                                    |
| `securityContext.readOnlyRootFilesystem`         | Whether the root file system is read-only in the Admin security context.                   | `true`                                    |
| `securityContext.allowPrivilegeEscalation`       | Whether the Admin security context allows privilege escalation.                            | `false`                                   |
| `podDisruptionBudget.enabled`                    | PodDisruptionBudget Status for Admin.                                                      | `false`                                   |
| `podDisruptionBudget.labels`                     | podDisruptionBudget Labels Definition for Admin.                                           | `~`                                       |
| `podDisruptionBudget.annotations`                | podDisruptionBudget Annotations Definition for Admin.                                      | `~`                                       |
| `podDisruptionBudget.minAvailable`               | podDisruptionBudget min Available Definition for Admin.                                    | `1`                                       |
| `podDisruptionBudget.maxUnavailable`             | podDisruptionBudget max Unavailable Definition for Admin.                                  | `1`                                       |
| `podDisruptionBudget.unhealthyPodEvictionPolicy` | podDisruptionBudget Unhealthy Pod Eviction Policy Definition for Admin.                    | `IfHealthyBudget`                         |
| `podSecurityPolicy.enabled`                      | Pod Security Policy Status for Admin.                                                      | `false`                                   |
| `podSecurityPolicy.labels`                       | Pod Security Policy Labels Definition for Admin.                                           | `~`                                       |
| `podSecurityPolicy.annotations`                  | Pod Security Policy Annotations Definition for Admin.                                      | `~`                                       |
| `networkPolicy.enabled`                          | NetworkPolicy Status for Admin.                                                            | `false`                                   |
| `networkPolicy.labels`                           | Network Policy Labels Definition for Admin.                                                | `~`                                       |
| `networkPolicy.annotations`                      | Network Policy Annotations Definition for Admin.                                           | `~`                                       |
| `networkPolicy.podSelector`                      | Network Policy Pod Selector Definition for Admin.                                          | `~`                                       |
| `networkPolicy.ingress`                          | Network Policy Ingress Definition for Admin.                                               | `~`                                       |
| `networkPolicy.egress`                           | Network Policy Egress Definition for Admin.                                                | `~`                                       |
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
| `zookeeper.namespaceOverride`                                 | NameSpace Override for ZooKeeper.                                                                                           | `~`                           |
| `zookeeper.labels`                                            | Labels for ZooKeeper.                                                                                                       | `~`                           |
| `zookeeper.annotations`                                       | Annotations for ZooKeeper.                                                                                                  | `~`                           |
| `zookeeper.clusterDomain`                                     | Cluster Domain Suffix for ZooKeeper.                                                                                        | `cluster.local`               |
| `zookeeper.replicas`                                          | Replica Deployment Count for ZooKeeper.                                                                                     | `1`                           |
| `zookeeper.image.registry`                                    | Image Name for ZooKeeper.                                                                                                   | `docker.io/bitnami/zookeeper` |
| `zookeeper.image.tag`                                         | Version Tag for ZooKeeper.                                                                                                  | `latest`                      |
| `zookeeper.image.pullPolicy`                                  | Pull Policy for ZooKeeper.                                                                                                  | `IfNotPresent`                |
| `zookeeper.securityContext.enabled`                           | security context Status for ZooKeeper.                                                                                      | `true`                        |
| `zookeeper.securityContext.fsGroup`                           | Non-privileged group identity identifier for Zookeeper security context.                                                    | `1001`                        |
| `zookeeper.containerSecurityContext.enabled`                  | container security context Status for ZooKeeper.                                                                            | `true`                        |
| `zookeeper.containerSecurityContext.runAsUser`                | Non-privileged user identifier for ZooKeeper container security context.                                                    | `1001`                        |
| `zookeeper.containerSecurityContext.runAsNonRoot`             | Whether the container security context for ZooKeeper runs as a non-privileged user.                                         | `true`                        |
| `zookeeper.containerSecurityContext.allowPrivilegeEscalation` | The state of privilege escalation in the Zookeeper container security context.                                              | `false`                       |
| `zookeeper.service.type`                                      | Define the service type for the ZooKeeper.                                                                                  | `ClusterIP`                   |
| `zookeeper.service.clusterIP`                                 | Define the service cluster IP for the ZooKeeper.                                                                            | `~`                           |
| `zookeeper.service.externalIPs`                               | Define the service external IP for the ZooKeeper.                                                                           | `~`                           |
| `zookeeper.service.loadBalancerIP`                            | Define the service loadBalancer IP for the ZooKeeper.                                                                       | `~`                           |
| `zookeeper.service.loadBalancerSourceRanges`                  | Define the service loadBalancer Source Ranges for the ZooKeeper.                                                            | `~`                           |
| `zookeeper.service.loadBalancerClass`                         | Define the service loadBalancer Class for the ZooKeeper.                                                                    | `~`                           |
| `zookeeper.service.sessionAffinity`                           | Define the session affinity strategy for the ZooKeeper service.                                                             | `None`                        |
| `zookeeper.service.publishNotReadyAddresses`                  | Define the publication of not-ready ZooKeeper service addresses to other components.                                        | `true`                        |
| `zookeeper.resources.limits.cpu`                              | Maximum Limit on CPU Resources for ZooKeeper.                                                                               | `128`                         |
| `zookeeper.resources.limits.memory`                           | Maximum Limit on Memory Resources for ZooKeeper.                                                                            | `128`                         |
| `zookeeper.resources.requests.cpu`                            | Maximum Request on CPU Resources for ZooKeeper.                                                                             | `128`                         |
| `zookeeper.resources.requests.memory`                         | Maximum Request on Memory Resources for ZooKeeper.                                                                          | `128`                         |
| `zookeeper.startupProbe.failureThreshold`                     | The allowed number of failures for the ZooKeeper container.                                                                 | `6`                           |
| `zookeeper.startupProbe.initialDelaySeconds`                  | Initialization Wait Time in Seconds After ZooKeeper Container Starts.                                                       | `30`                          |
| `zookeeper.startupProbe.periodSeconds`                        | The ZooKeeper container periodically checks availability.                                                                   | `10`                          |
| `zookeeper.startupProbe.successThreshold`                     | The success threshold for the ZooKeeper container.                                                                          | `1`                           |
| `zookeeper.startupProbe.timeoutSeconds`                       | Response Timeout Duration in Seconds After ZooKeeper Container Starts.                                                      | `5`                           |
| `zookeeper.startupProbe.exec.command`                         | Define the health check command for the ZooKeeper container.                                                                | `~`                           |
| `zookeeper.readinessProbe.failureThreshold`                   | The allowed number of failures for the ZooKeeper container.                                                                 | `6`                           |
| `zookeeper.readinessProbe.initialDelaySeconds`                | Initialization Wait Time in Seconds After ZooKeeper Container Starts.                                                       | `30`                          |
| `zookeeper.readinessProbe.periodSeconds`                      | The ZooKeeper container periodically checks availability.                                                                   | `10`                          |
| `zookeeper.readinessProbe.successThreshold`                   | The success threshold for the ZooKeeper container.                                                                          | `1`                           |
| `zookeeper.readinessProbe.timeoutSeconds`                     | Response Timeout Duration in Seconds After ZooKeeper Container Starts.                                                      | `5`                           |
| `zookeeper.readinessProbe.exec.command`                       | Define the health check command for the ZooKeeper container.                                                                | `~`                           |
| `zookeeper.livenessProbe.failureThreshold`                    | The allowed number of failures for the ZooKeeper container.                                                                 | `6`                           |
| `zookeeper.livenessProbe.initialDelaySeconds`                 | Initialization Wait Time in Seconds After ZooKeeper Container Starts.                                                       | `30`                          |
| `zookeeper.livenessProbe.periodSeconds`                       | The ZooKeeper container periodically checks availability.                                                                   | `10`                          |
| `zookeeper.livenessProbe.successThreshold`                    | The success threshold for the ZooKeeper container.                                                                          | `1`                           |
| `zookeeper.livenessProbe.timeoutSeconds`                      | Response Timeout Duration in Seconds After ZooKeeper Container Starts.                                                      | `5`                           |
| `zookeeper.livenessProbe.exec.command`                        | Define the health check command for the ZooKeeper container.                                                                | `~`                           |
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
| `nacos.namespaceOverride`                                 | NameSpace Override for Nacos.                                                                               | `~`                                                                                             |
| `nacos.labels`                                            | Labels for Nacos.                                                                                           | `~`                                                                                             |
| `nacos.annotations`                                       | Annotations for Nacos.                                                                                      | `~`                                                                                             |
| `nacos.clusterDomain`                                     | Cluster Domain Suffix for Nacos.                                                                            | `cluster.local`                                                                                 |
| `nacos.replicas`                                          | Replica Deployment Count for Nacos.                                                                         | `1`                                                                                             |
| `nacos.plugin.enabled`                                    | Plugin Status for Nacos.                                                                                    | `true`                                                                                          |
| `nacos.plugin.image.registry`                             | Plugin Image Name for Nacos.                                                                                | `nacos/nacos-peer-finder-plugin`                                                                |
| `nacos.plugin.image.tag`                                  | Plugin Version Tag for Nacos.                                                                               | `1.1`                                                                                           |
| `nacos.plugin.image.pullPolicy`                           | Plugin Pull Policy for Nacos.                                                                               | `IfNotPresent`                                                                                  |
| `nacos.image.registry`                                    | Image Name for Nacos.                                                                                       | `docker.io/nacos/nacos-server`                                                                  |
| `nacos.image.tag`                                         | Version Tag for Nacos.                                                                                      | `latest`                                                                                        |
| `nacos.image.pullPolicy`                                  | Pull Policy for Nacos.                                                                                      | `IfNotPresent`                                                                                  |
| `nacos.securityContext.enabled`                           | security context Status for Nacos.                                                                          | `true`                                                                                          |
| `nacos.securityContext.fsGroup`                           | Non-privileged group identity identifier for Nacos security context.                                        | `1001`                                                                                          |
| `nacos.containerSecurityContext.enabled`                  | container security context Status for Nacos.                                                                | `true`                                                                                          |
| `nacos.containerSecurityContext.runAsUser`                | Non-privileged user identifier for Nacos container security context.                                        | `1001`                                                                                          |
| `nacos.containerSecurityContext.runAsNonRoot`             | Whether the container security context for Nacos runs as a non-privileged user.                             | `true`                                                                                          |
| `nacos.containerSecurityContext.allowPrivilegeEscalation` | The state of privilege escalation in the Nacos container security context.                                  | `false`                                                                                         |
| `nacos.service.type`                                      | Define the service type for the Nacos.                                                                      | `NodePort`                                                                                      |
| `nacos.service.clusterIP`                                 | Define the service cluster IP for the Nacos.                                                                | `~`                                                                                             |
| `nacos.service.externalIPs`                               | Define the service external IP for the Nacos.                                                               | `~`                                                                                             |
| `nacos.service.loadBalancerIP`                            | Define the service loadBalancer IP for the Nacos.                                                           | `~`                                                                                             |
| `nacos.service.loadBalancerSourceRanges`                  | Define the service loadBalancer Source Ranges for the Nacos.                                                | `~`                                                                                             |
| `nacos.service.loadBalancerClass`                         | Define the service loadBalancer Class for the Nacos.                                                        | `~`                                                                                             |
| `nacos.service.sessionAffinity`                           | Define the session affinity strategy for the Nacos service.                                                 | `None`                                                                                          |
| `nacos.service.publishNotReadyAddresses`                  | Define the publication of not-ready Nacos service addresses to other components.                            | `true`                                                                                          |
| `nacos.startupProbe.initialDelaySeconds`                  | Initialization Wait Time in Seconds After Nacos Container Starts.                                           | `180`                                                                                           |
| `nacos.startupProbe.periodSeconds`                        | The Nacos container periodically checks availability.                                                       | `5`                                                                                             |
| `nacos.startupProbe.timeoutSeconds`                       | Response Timeout Duration in Seconds After Nacos Container Starts.                                          | `10`                                                                                            |
| `nacos.startupProbe.httpGet.scheme`                       | Define the network protocol used for health check probe requests when the container starts.                 | `HTTP`                                                                                          |
| `nacos.startupProbe.httpGet.port`                         | Checking the container with an HTTP GET request to a target port.                                           | `8848`                                                                                          |
| `nacos.startupProbe.httpGet.path`                         | Checking the container with an HTTP GET request to a target path.                                           | `/nacos/v1/console/health/readiness`                                                            |
| `nacos.readinessProbe.initialDelaySeconds`                | Initialization Wait Time in Seconds After Nacos Container Starts.                                           | `180`                                                                                           |
| `nacos.readinessProbe.periodSeconds`                      | The Nacos container periodically checks availability.                                                       | `5`                                                                                             |
| `nacos.readinessProbe.timeoutSeconds`                     | Response Timeout Duration in Seconds After Nacos Container Starts.                                          | `10`                                                                                            |
| `nacos.readinessProbe.httpGet.scheme`                     | Define the network protocol used for health check probe requests when the container starts.                 | `HTTP`                                                                                          |
| `nacos.readinessProbe.httpGet.port`                       | Checking the container with an HTTP GET request to a target port.                                           | `8848`                                                                                          |
| `nacos.readinessProbe.httpGet.path`                       | Checking the container with an HTTP GET request to a target path.                                           | `/nacos/v1/console/health/readiness`                                                            |
| `nacos.livenessProbe.initialDelaySeconds`                 | Initialization Wait Time in Seconds After Nacos Container Starts.                                           | `180`                                                                                           |
| `nacos.livenessProbe.periodSeconds`                       | The Nacos container periodically checks availability.                                                       | `5`                                                                                             |
| `nacos.livenessProbe.timeoutSeconds`                      | Response Timeout Duration in Seconds After Nacos Container Starts.                                          | `10`                                                                                            |
| `nacos.livenessProbe.httpGet.scheme`                      | Define the network protocol used for health check probe requests when the container starts.                 | `HTTP`                                                                                          |
| `nacos.livenessProbe.httpGet.port`                        | Checking the container with an HTTP GET request to a target port.                                           | `8848`                                                                                          |
| `nacos.livenessProbe.httpGet.path`                        | Checking the container with an HTTP GET request to a target path.                                           | `/nacos/v1/console/health/readiness`                                                            |
| `nacos.resources.limits.cpu`                              | Maximum Limit on CPU Resources for Nacos.                                                                   | `128`                                                                                           |
| `nacos.resources.limits.memory`                           | Maximum Limit on Memory Resources for Nacos.                                                                | `128`                                                                                           |
| `nacos.resources.requests.cpu`                            | Maximum Request on CPU Resources for Nacos.                                                                 | `128`                                                                                           |
| `nacos.resources.requests.memory`                         | Maximum Request on Memory Resources for Nacos.                                                              | `128`                                                                                           |
| `nacos.serverPort`                                        | Define the service port for the Nacos.                                                                      | `8848`                                                                                          |
| `nacos.preferhostmode`                                    | Enable Nacos cluster node domain name support                                                               | `~`                                                                                             |
| `nacos.storage.type`                                      | Nacos data storage method `mysql` or `embedded`. The `embedded` supports either standalone or cluster mode. | `embedded`                                                                                      |
| `nacos.storage.db.host`                                   | Specify the database host for Nacos storing configuration data.                                             | `localhost`                                                                                     |
| `nacos.storage.db.name`                                   | Specify the database name for Nacos storing configuration data.                                             | `nacos`                                                                                         |
| `nacos.storage.db.port`                                   | Specify the database port for Nacos storing configuration data.                                             | `3306`                                                                                          |
| `nacos.storage.db.username`                               | Specify the database username for Nacos storing configuration data.                                         | `mysql`                                                                                         |
| `nacos.storage.db.password`                               | Specify the database password for Nacos storing configuration data.                                         | `passw0rd`                                                                                      |
| `nacos.storage.db.param`                                  | Specify the database url parameter for Nacos storing configuration data.                                    | `characterEncoding=utf8&connectTimeout=1000&socketTimeout=3000&autoReconnect=true&useSSL=false` |

### Ingress

| Key                                                         | Description                                                                                             | Default               |
|-------------------------------------------------------------|---------------------------------------------------------------------------------------------------------|-----------------------|
| `ingress.enabled`                                           | Enable Ingress Status.                                                                                  | `true`                |
| `ingress.hosts.admin`                                       | Define the domain name for the admin application host.                                                  | `admin.k8s.example`   |
| `ingress.hosts.prometheus`                                  | Define the domain name for the prometheus application host.                                             | `prom.k8s.example`    |
| `ingress.hosts.grafana`                                     | Define the domain name for the grafana application host.                                                | `grafana.k8s.example` |
| `ingress.nameOverride`                                      | Name Override for Ingress.                                                                              | `~`                   |
| `ingress.namespaceOverride`                                 | NameSpace Override Ingress.                                                                             | `~`                   |
| `ingress.labels`                                            | Labels for  Ingress.                                                                                    | `~`                   |       
| `ingress.annotations`                                       | Annotations for Ingress.                                                                                | `~`                   |       
| `ingress.nodeSelector`                                      | Node Scheduling for Ingress.                                                                            | `~`                   |      
| `ingress.replicas`                                          | Replica Deployment Count for Ingress.                                                                   | `1`                   |
| `ingress.image.registry`                                    | Image Name for Ingress.                                                                                 | `docker.io/traefik`   |
| `ingress.image.tag`                                         | Version Tag for Ingress.                                                                                | `v2.10.4`             |
| `ingress.image.pullPolicy`                                  | Pull Policy for Ingress.                                                                                | `IfNotPresent`        |
| `ingress.readinessProbe.failureThreshold`                   | The allowed number of failures for the Ingress container.                                               | `1`                   |
| `ingress.readinessProbe.initialDelaySeconds`                | Initialization Wait Time in Seconds After Ingress Container Starts.                                     | `2`                   |
| `ingress.readinessProbe.periodSeconds`                      | The Ingress container periodically checks availability.                                                 | `10`                  |
| `ingress.readinessProbe.successThreshold`                   | The success threshold for the Ingress container.                                                        | `1`                   |
| `ingress.readinessProbe.timeoutSeconds`                     | Response Timeout Duration in Seconds After Ingress Container Starts.                                    | `2`                   |
| `ingress.httpGet.path`                                      | Checking the container with an HTTP GET request to a target path.                                       | `/ping`               |
| `ingress.httpGet.port`                                      | Checking the container with an HTTP GET request to a target port.                                       | `9000`                |
| `ingress.httpGet.scheme`                                    | Define the network protocol used for health check probe requests when the container starts.             | `HTTP`                |
| `ingress.livenessProbe.failureThreshold`                    | The allowed number of failures for the Ingress container.                                               | `3`                   |
| `ingress.livenessProbe.initialDelaySeconds`                 | Initialization Wait Time in Seconds After Ingress Container Starts.                                     | `2`                   |
| `ingress.livenessProbe.periodSeconds`                       | The Ingress container periodically checks availability.                                                 | `10`                  |
| `ingress.livenessProbe.successThreshold`                    | The success threshold for the Ingress container.                                                        | `1`                   |
| `ingress.livenessProbe.timeoutSeconds`                      | Response Timeout Duration in Seconds After Ingress Container Starts.                                    | `2`                   |
| `ingress.httpGet.path`                                      | Checking the container with an HTTP GET request to a target path.                                       | `/ping`               |
| `ingress.httpGet.port`                                      | Checking the container with an HTTP GET request to a target port.                                       | `9000`                |
| `ingress.httpGet.scheme`                                    | Define the network protocol used for health check probe requests when the container starts.             | `HTTP`                |
| `ingress.strategy.rollingUpdate.maxSurge`                   | Ingress Update Strategy Expected Replicas.                                                              | `1`                   |
| `ingress.strategy.rollingUpdate.maxUnavailable`             | Ingress Update Strategy Maximum Unavailable Replicas.                                                   | `0`                   |
| `ingress.securityContext.runAsUser`                         | Non-privileged user identifier for Ingress security context.                                            | `65532`               |
| `ingress.securityContext.runAsGroup`                        | Non-privileged group identity identifier for Ingress security context.                                  | `65532`               |
| `ingress.securityContext.runAsNonRoot`                      | Whether the security context for Ingress runs as a non-privileged user.                                 | `true`                |
| `ingress.containersecurityContext.capabilities.drop`        | Whether Linux kernel capabilities or permissions are enabled in the Ingress container security context. | `[ALL]`               |
| `ingress.containersecurityContext.readOnlyRootFilesystem`   | Whether the root file system is read-only in the Ingress container security context.                    | `true`                |
| `ingress.containersecurityContext.allowPrivilegeEscalation` | Whether the Ingress container security context allows privilege escalation.                             | `false`               |
| `ingress.resources.limits.cpu`                              | Maximum Limit on CPU Resources for Ingress.                                                             | `128`                 |
| `ingress.resources.limits.memory`                           | Maximum Limit on Memory Resources for Ingress.                                                          | `128`                 |
| `ingress.resources.requests.cpu`                            | Maximum Request on CPU Resources for Ingress.                                                           | `128`                 |
| `ingress.resources.requests.memory`                         | Maximum Request on Memory Resources for Ingress.                                                        | `128`                 |

### Jobs

| Key                      | Description              | Default                     |
|--------------------------|--------------------------|-----------------------------|
| `jobs.namespaceOverride` | NameSpace Override Jobs. | `~`                         |
| `jobs.labels`            | Labels for Jobs.         | `~`                         |       
| `jobs.annotations`       | Annotations for Jobs.    | `~`                         |
| `jobs.image.registry`    | Image Name for Jobs.     | `docker.io/bitnami/kubectl` |
| `jobs.image.tag`         | Version Tag for Jobs.    | `1.28.4`                    |
| `jobs.image.pullPolicy`  | Pull Policy for Jobs.    | `IfNotPresent`              |


