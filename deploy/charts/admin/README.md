# Admin         

![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

The Helm control chart for Admin.

## Values

### Admin

| Key                                              | Description                                                                      | Default                        |
|--------------------------------------------------|----------------------------------------------------------------------------------|--------------------------------|
| `deployType`                                     |                                                                                  | `Deployment`                   |
| `nameOverride`                                   | Name Override for Admin.                                                         | `{}`                           |
| `namespaceOverride`                              | NameSpace Override for Admin.                                                    | `{}`                           |
| `labels`                                         | Labels for Admin.                                                                | `{}`                           |
| `annotations`                                    | Annotations for Admin.                                                           | `{}`                           |
| `nodeSelector`                                   | Node Scheduling for Admin.                                                       | `{}`                           |
| `imagePullSecrets`                               | Image Pull Credentials for Admin.                                                | `[]`                           |
| `clusterDomain`                                  | Cluster Domain Suffix for Admin.                                                 | `cluster.local`                |
| `replicas`                                       | Replica Deployment Count for Admin.                                              | `1`                            |
| `image.registry`                                 | Image Name for Admin.                                                            | `docker.io/apache/dubbo-admin` |
| `image.tag`                                      | Version Tag for Admin.                                                           | `latest`                       |
| `image.pullPolicy`                               | Pull Policy for Admin.                                                           | `IfNotPresent`                 |
| `rbac.enabled`                                   | Role-Based Access Control Status for Admin.                                      | `true`                         |
| `rbac.labels`                                    | Role-Based Access Control Labels Definition for Admin.                           | `{}`                           |
| `rbac.annotations`                               | Role-Based Access Control Annotations Definition for Admin.                      | `{}`                           |
| `serviceAccount.enabled`                         | Service Accounts Status for Admin.                                               | `true`                         |
| `serviceAccount.labels`                          | Service Accounts Labels Definition for Admin.                                    | `{}`                           |
| `serviceAccount.annotations`                     | Service Accounts Annotations Definition for Admin.                               | `{}`                           |
| `volumeMounts`                                   | Internal Mount Directory for Admin.                                              | `[]`                           |
| `volumes`                                        | External Mount Directory for Admin.                                              | `[]`                           |
| `configMap`                                      | ConfigMap Mount Configuration for Admin.                                         | `{}`                           |
| `secret`                                         | Secret Mount Configuration for Admin.                                            | `{}`                           |
| `strategy.type`                                  | Policy Type for Admin.                                                           | `RollingUpdate`                |
| `strategy.rollingUpdate.maxSurge`                | Admin Update Strategy Expected Replicas.                                         | `25%`                          |
| `strategy.rollingUpdate.maxUnavailable`          | Admin Update Strategy Maximum Unavailable Replicas.                              | `1`                            |
| `updateStrategy.type`                            | Update Policy Type for Admin.                                                    | `RollingUpdate`                |
| `updateStrategy.rollingUpdate`                   | Rolling Update Strategy for Admin.                                               | `{}`                           |
| `minReadySeconds`                                | Seconds to Wait Before Admin Readiness.                                          | `0`                            |
| `revisionHistoryLimit`                           | Number of Revision Versions Saved in History for Admin.                          | `10`                           |
| `terminationGracePeriodSeconds`                  | Graceful Termination Seconds for Admin Hooks.                                    | `30`                           |
| `startupProbe.initialDelaySeconds`               | Initialization Wait Time in Seconds After Admin Container Starts.                | `60`                           |
| `startupProbe.timeoutSeconds`                    | Response Timeout Duration in Seconds After Admin Container Starts.               | `30`                           |
| `startupProbe.periodSeconds`                     | The Admin container periodically checks availability.                            | `10`                           |
| `startupProbe.successThreshold`                  | The success threshold for the Admin container.                                   | `1`                            |
| `startupProbe.httpGet.path`                      | Checking the container with an HTTP GET request to a target path.                | `/`                            |
| `startupProbe.httpGet.port`                      | Checking the container with an HTTP GET request to a target port.                | `8080`                         |
| `readinessProbe.initialDelaySeconds`             | Initialization Wait Time in Seconds After Admin Container Starts.                | `60`                           |
| `readinessProbe.timeoutSeconds`                  | Response Timeout Duration in Seconds After Admin Container Starts.               | `30`                           |
| `readinessProbe.periodSeconds`                   | The Admin container periodically checks availability.                            | `10`                           |
| `readinessProbe.successThreshold`                | The success threshold for the Admin container.                                   | `1`                            |
| `readinessProbe.httpGet.path`                    | Checking the container with an HTTP GET request to a target path.                | `/`                            |
| `readinessProbe.httpGet.port`                    | Checking the container with an HTTP GET request to a target port.                | `8080`                         |
| `livenessProbe.initialDelaySeconds`              | Initialization Wait Time in Seconds After Admin Container Starts.                | `60`                           |
| `livenessProbe.timeoutSeconds`                   | Response Timeout Duration in Seconds After Admin Container Starts.               | `30`                           |
| `livenessProbe.periodSeconds`                    | The Admin container periodically checks availability.                            | `10`                           |
| `livenessProbe.successThreshold`                 | The success threshold for the Admin container.                                   | `1`                            |
| `livenessProbe.httpGet.path`                     | Checking the container with an HTTP GET request to a target path.                | `/`                            |
| `livenessProbe.httpGet.port`                     | Checking the container with an HTTP GET request to a target port.                | `8080`                         |
| `lifecycleHooks`                                 | Graceful hooks for the Admin.                                                    | `[]`                           |
| `service.enabled`                                | Service Status for Admin.                                                        | `true`                         |
| `service.labels`                                 | Service Label Definition for Admin.                                              | `{}`                           |
| `service.annotations`                            | Service Annotations Definition for Admin.                                        | `{}`                           |
| `service.type`                                   | Define the service type for the Admin.                                           | `ClusterIP`                    |
| `service.clusterIP`                              | Define the service cluster IP for the Admin.                                     | `~`                            |
| `service.externalIPs`                            | Define the service external IP for the Admin.                                    | `~`                            |
| `service.loadBalancerIP`                         | Define the service loadBalancer IP for the Admin.                                | `~`                            |
| `service.loadBalancerSourceRanges`               | Define the service loadBalancer Source Ranges for the Admin.                     | `~`                            |
| `service.loadBalancerClass`                      | Define the service loadBalancer Class for the Admin.                             | `~`                            |
| `service.sessionAffinity`                        | Define the session affinity strategy for the Admin service.                      | `None`                         |
| `service.publishNotReadyAddresses`               | Define the publication of not-ready Admin service addresses to other components. | `true`                         |
| `service.protocol`                               | Service Protocol Definition for Admin.                                           | `TCP`                          |
| `resources.limits.cpu`                           | Maximum Limit on CPU Resources for Admin.                                        | `128`                          |
| `resources.limits.memory`                        | Maximum Limit on Memory Resources for Admin.                                     | `128`                          |
| `resources.requests.cpu`                         | Maximum Request on CPU Resources for Admin.                                      | `128`                          |
| `resources.requests.memory`                      | Maximum Request on Memory Resources for Admin.                                   | `128`                          |
| `tolerations`                                    | toleration's Definition for Admin.                                               | `[]`                           |
| `persistence.enabled`                            | Persistence Status for Admin.                                                    | `false`                        | 
| `persistence.labels`                             | Persistence Labels Definition for Admin.                                         | `{}`                           |
| `persistence.annotations`                        | Persistence Annotations Definition for Admin.                                    | `{}`                           |
| `persistence.claimName`                          | Persistence claim name Definition for Admin.                                     | `""`                           | 
| `persistence.storageclass`                       | Persistence storage class Definition for Admin.                                  | `""`                           | 
| `persistence.size`                               | Persistence size Definition for Admin.                                           | `1Gi`                          |
| `persistence.accessModes`                        | Persistence accessModes Definition for Admin.                                    | `ReadWriteOnce`                | 
| `securityContext.runAsNonRoot`                   | Whether the security context for Admin runs as a non-privileged user.            | `false`                        |
| `securityContext.runAsUser`                      | Non-privileged user identifier for Admin security context.                       | `1000`                         |
| `securityContext.runAsGroup`                     | Non-privileged group identity identifier for Admin security context.             | `1000`                         |
| `securityContext.readOnlyRootFilesystem`         | Whether the root file system is read-only in the Admin security context.         | `true`                         |
| `securityContext.allowPrivilegeEscalation`       | Whether the Admin security context allows privilege escalation.                  | `false`                        |
| `podDisruptionBudget.enabled`                    | PodDisruptionBudget Status for Admin.                                            | `false`                        |
| `podDisruptionBudget.labels`                     | podDisruptionBudget Labels Definition for Admin.                                 | `{}`                           |
| `podDisruptionBudget.annotations`                | podDisruptionBudget Annotations Definition for Admin.                            | `{}`                           |
| `podDisruptionBudget.minAvailable`               | podDisruptionBudget min Available Definition for Admin.                          | `1`                            |
| `podDisruptionBudget.maxUnavailable`             | podDisruptionBudget max Unavailable Definition for Admin.                        | `1`                            |
| `podDisruptionBudget.unhealthyPodEvictionPolicy` | podDisruptionBudget Unhealthy Pod Eviction Policy Definition for Admin.          | `IfHealthyBudget`              |
| `podSecurityPolicy.enabled`                      | Pod Security Policy Status for Admin.                                            | `false`                        |
| `podSecurityPolicy.labels`                       | Pod Security Policy Labels Definition for Admin.                                 | `{}`                           |
| `podSecurityPolicy.annotations`                  | Pod Security Policy Annotations Definition for Admin.                            | `{}`                           |
| `networkPolicy.enabled`                          | NetworkPolicy Status for Admin.                                                  | `false`                        |
| `networkPolicy.labels`                           | Network Policy Labels Definition for Admin.                                      | `{}`                           |
| `networkPolicy.annotations`                      | Network Policy Annotations Definition for Admin.                                 | `{}`                           |
| `networkPolicy.podSelector`                      | Network Policy Pod Selector Definition for Admin.                                | `{}`                           |
| `networkPolicy.ingress`                          | Network Policy Ingress Definition for Admin.                                     | `[]`                           |
| `networkPolicy.egress`                           | Network Policy Egress Definition for Admin.                                      | `[]`                           |

## ZooKeeper

| Key                                                           | Description                                                                          | Default                       |
|---------------------------------------------------------------|--------------------------------------------------------------------------------------|-------------------------------|
| `zookeeper.enabled`                                           | ZooKeeper Status for Admin.                                                          | `true`                        |
| `zookeeper.nameOverride`                                      | Name Override for ZooKeeper.                                                         | `{}`                          |
| `zookeeper.namespaceOverride`                                 | NameSpace Override for ZooKeeper.                                                    | `{}`                          |
| `zookeeper.labels`                                            | Labels for ZooKeeper.                                                                | `{}`                          |
| `zookeeper.annotations`                                       | Annotations for ZooKeeper.                                                           | `{}`                          |
| `zookeeper.clusterDomain`                                     | Cluster Domain Suffix for ZooKeeper.                                                 | `cluster.local`               |
| `zookeeper.replicas`                                          | Replica Deployment Count for ZooKeeper.                                              | `1`                           |
| `zookeeper.image.registry`                                    | Image Name for ZooKeeper.                                                            | `docker.io/bitnami/zookeeper` |
| `zookeeper.image.tag`                                         | Version Tag for ZooKeeper.                                                           | `3.8.1-debian-11-r18`         |
| `zookeeper.image.pullPolicy`                                  | Pull Policy for ZooKeeper.                                                           | `IfNotPresent`                |
| `zookeeper.securityContext.enabled`                           | security context Status for ZooKeeper.                                               | `true`                        |
| `zookeeper.securityContext.fsGroup`                           | Non-privileged group identity identifier for Zookeeper security context.             | `1001`                        |
| `zookeeper.containerSecurityContext.enabled`                  | container security context Status for ZooKeeper.                                     | `true`                        |
| `zookeeper.containerSecurityContext.runAsUser`                | Non-privileged user identifier for ZooKeeper container security context.             | `1001`                        |
| `zookeeper.containerSecurityContext.runAsNonRoot`             | Whether the container security context for ZooKeeper runs as a non-privileged user.  | `true`                        |
| `zookeeper.containerSecurityContext.allowPrivilegeEscalation` | The state of privilege escalation in the Zookeeper container security context.       | `false`                       |
| `zookeeper.service.type`                                      | Define the service type for the ZooKeeper.                                           | `ClusterIP`                   |
| `zookeeper.service.clusterIP`                                 | Define the service cluster IP for the ZooKeeper.                                     | `~`                           |
| `zookeeper.service.externalIPs`                               | Define the service external IP for the ZooKeeper.                                    | `~`                           |
| `zookeeper.service.loadBalancerIP`                            | Define the service loadBalancer IP for the ZooKeeper.                                | `~`                           |
| `zookeeper.service.loadBalancerSourceRanges`                  | Define the service loadBalancer Source Ranges for the ZooKeeper.                     | `~`                           |
| `zookeeper.service.loadBalancerClass`                         | Define the service loadBalancer Class for the ZooKeeper.                             | `~`                           |
| `zookeeper.service.sessionAffinity`                           | Define the session affinity strategy for the ZooKeeper service.                      | `None`                        |
| `zookeeper.service.publishNotReadyAddresses`                  | Define the publication of not-ready ZooKeeper service addresses to other components. | `true`                        |
| `zookeeper.resources.limits.cpu`                              | Maximum Limit on CPU Resources for ZooKeeper.                                        | `128`                         |
| `zookeeper.resources.limits.memory`                           | Maximum Limit on Memory Resources for ZooKeeper.                                     | `128`                         |
| `zookeeper.resources.requests.cpu`                            | Maximum Request on CPU Resources for ZooKeeper.                                      | `128`                         |
| `zookeeper.resources.requests.memory`                         | Maximum Request on Memory Resources for ZooKeeper.                                   | `128`                         |
| `zookeeper.startupProbe.failureThreshold`                     | The allowed number of failures for the ZooKeeper container.                          | `6`                           |
| `zookeeper.startupProbe.initialDelaySeconds`                  | Initialization Wait Time in Seconds After ZooKeeper Container Starts.                | `30`                          |
| `zookeeper.startupProbe.periodSeconds`                        | The ZooKeeper container periodically checks availability.                            | `10`                          |
| `zookeeper.startupProbe.successThreshold`                     | The success threshold for the ZooKeeper container.                                   | `1`                           |
| `zookeeper.startupProbe.timeoutSeconds`                       | Response Timeout Duration in Seconds After ZooKeeper Container Starts.               | `5`                           |
| `zookeeper.startupProbe.exec.command`                         | Define the health check command for the ZooKeeper container.                         | `[]`                          |
| `zookeeper.readinessProbe.failureThreshold`                   | The allowed number of failures for the ZooKeeper container.                          | `6`                           |
| `zookeeper.readinessProbe.initialDelaySeconds`                | Initialization Wait Time in Seconds After ZooKeeper Container Starts.                | `30`                          |
| `zookeeper.readinessProbe.periodSeconds`                      | The ZooKeeper container periodically checks availability.                            | `10`                          |
| `zookeeper.readinessProbe.successThreshold`                   | The success threshold for the ZooKeeper container.                                   | `1`                           |
| `zookeeper.readinessProbe.timeoutSeconds`                     | Response Timeout Duration in Seconds After ZooKeeper Container Starts.               | `5`                           |
| `zookeeper.readinessProbe.exec.command`                       | Define the health check command for the ZooKeeper container.                         | `[]`                          |
| `zookeeper.livenessProbe.failureThreshold`                    | The allowed number of failures for the ZooKeeper container.                          | `6`                           |
| `zookeeper.livenessProbe.initialDelaySeconds`                 | Initialization Wait Time in Seconds After ZooKeeper Container Starts.                | `30`                          |
| `zookeeper.livenessProbe.periodSeconds`                       | The ZooKeeper container periodically checks availability.                            | `10`                          |
| `zookeeper.livenessProbe.successThreshold`                    | The success threshold for the ZooKeeper container.                                   | `1`                           |
| `zookeeper.livenessProbe.timeoutSeconds`                      | Response Timeout Duration in Seconds After ZooKeeper Container Starts.               | `5`                           |
| `zookeeper.livenessProbe.exec.command`                        | Define the health check command for the ZooKeeper container.                         | `[]`                          |

### Nacos

| Key                                                       | Description                                                                                 | Default                              |
|-----------------------------------------------------------|---------------------------------------------------------------------------------------------|--------------------------------------|
| `nacos.enabled`                                           | Nacos Status for Admin.                                                                     | `false`                              |
| `nacos.mode`                                              | Run Mode standalone or cluster.                                                             | `standalone`                         |
| `nacos.nameOverride`                                      | Name Override for Nacos.                                                                    | `{}`                                 |
| `nacos.namespaceOverride`                                 | NameSpace Override for Nacos.                                                               | `{}`                                 |
| `nacos.labels`                                            | Labels for Nacos.                                                                           | `{}`                                 |
| `nacos.annotations`                                       | Annotations for Nacos.                                                                      | `{}`                                 |
| `nacos.clusterDomain`                                     | Cluster Domain Suffix for Nacos.                                                            | `cluster.local`                      |
| `nacos.replicas`                                          | Replica Deployment Count for Nacos.                                                         | `1`                                  |
| `nacos.plugin.enabled`                                    | Plugin Status for Nacos.                                                                    | `true`                               |
| `nacos.plugin.image.registry`                             | Plugin Image Name for Nacos.                                                                | `nacos/nacos-peer-finder-plugin`     |
| `nacos.plugin.image.tag`                                  | Plugin Version Tag for Nacos.                                                               | `1.1`                                |
| `nacos.plugin.image.pullPolicy`                           | Plugin Pull Policy for Nacos.                                                               | `IfNotPresent`                       |
| `nacos.image.registry`                                    | Image Name for Nacos.                                                                       | `docker.io/nacos/nacos-server`       |
| `nacos.image.tag`                                         | Version Tag for Nacos.                                                                      | `latest`                             |
| `nacos.image.pullPolicy`                                  | Pull Policy for Nacos.                                                                      | `IfNotPresent`                       |
| `nacos.securityContext.enabled`                           | security context Status for Nacos.                                                          | `true`                               |
| `nacos.securityContext.fsGroup`                           | Non-privileged group identity identifier for Nacos security context.                        | `1001`                               |
| `nacos.containerSecurityContext.enabled`                  | container security context Status for Nacos.                                                | `true`                               |
| `nacos.containerSecurityContext.runAsUser`                | Non-privileged user identifier for Nacos container security context.                        | `1001`                               |
| `nacos.containerSecurityContext.runAsNonRoot`             | Whether the container security context for Nacos runs as a non-privileged user.             | `true`                               |
| `nacos.containerSecurityContext.allowPrivilegeEscalation` | The state of privilege escalation in the Nacos container security context.                  | `false`                              |
| `nacos.service.type`                                      | Define the service type for the Nacos.                                                      | `NodePort`                           |
| `nacos.service.clusterIP`                                 | Define the service cluster IP for the Nacos.                                                | `~`                                  |
| `nacos.service.externalIPs`                               | Define the service external IP for the Nacos.                                               | `~`                                  |
| `nacos.service.loadBalancerIP`                            | Define the service loadBalancer IP for the Nacos.                                           | `~`                                  |
| `nacos.service.loadBalancerSourceRanges`                  | Define the service loadBalancer Source Ranges for the Nacos.                                | `~`                                  |
| `nacos.service.loadBalancerClass`                         | Define the service loadBalancer Class for the Nacos.                                        | `~`                                  |
| `nacos.service.sessionAffinity`                           | Define the session affinity strategy for the Nacos service.                                 | `None`                               |
| `nacos.service.publishNotReadyAddresses`                  | Define the publication of not-ready Nacos service addresses to other components.            | `true`                               |
| `nacos.startupProbe.initialDelaySeconds`                  | Initialization Wait Time in Seconds After Nacos Container Starts.                           | `180`                                |
| `nacos.startupProbe.periodSeconds`                        | The Nacos container periodically checks availability.                                       | `5`                                  |
| `nacos.startupProbe.timeoutSeconds`                       | Response Timeout Duration in Seconds After Nacos Container Starts.                          | `10`                                 |
| `nacos.startupProbe.httpGet.scheme`                       | Define the network protocol used for health check probe requests when the container starts. | `HTTP`                               |
| `nacos.startupProbe.httpGet.port`                         | Checking the container with an HTTP GET request to a target port.                           | `8848`                               |
| `nacos.startupProbe.httpGet.path`                         | Checking the container with an HTTP GET request to a target path.                           | `/nacos/v1/console/health/readiness` |
| `nacos.readinessProbe.initialDelaySeconds`                | Initialization Wait Time in Seconds After Nacos Container Starts.                           | `180`                                |
| `nacos.readinessProbe.periodSeconds`                      | The Nacos container periodically checks availability.                                       | `5`                                  |
| `nacos.readinessProbe.timeoutSeconds`                     | Response Timeout Duration in Seconds After Nacos Container Starts.                          | `10`                                 |
| `nacos.readinessProbe.httpGet.scheme`                     | Define the network protocol used for health check probe requests when the container starts. | `HTTP`                               |
| `nacos.readinessProbe.httpGet.port`                       | Checking the container with an HTTP GET request to a target port.                           | `8848`                               |
| `nacos.readinessProbe.httpGet.path`                       | Checking the container with an HTTP GET request to a target path.                           | `/nacos/v1/console/health/readiness` |
| `nacos.livenessProbe.initialDelaySeconds`                 | Initialization Wait Time in Seconds After Nacos Container Starts.                           | `180`                                |
| `nacos.livenessProbe.periodSeconds`                       | The Nacos container periodically checks availability.                                       | `5`                                  |
| `nacos.livenessProbe.timeoutSeconds`                      | Response Timeout Duration in Seconds After Nacos Container Starts.                          | `10`                                 |
| `nacos.livenessProbe.httpGet.scheme`                      | Define the network protocol used for health check probe requests when the container starts. | `HTTP`                               |
| `nacos.livenessProbe.httpGet.port`                        | Checking the container with an HTTP GET request to a target port.                           | `8848`                               |
| `nacos.livenessProbe.httpGet.path`                        | Checking the container with an HTTP GET request to a target path.                           | `/nacos/v1/console/health/readiness` |
| `nacos.resources.limits.cpu`                              | Maximum Limit on CPU Resources for Nacos.                                                   | `128`                                |
| `nacos.resources.limits.memory`                           | Maximum Limit on Memory Resources for Nacos.                                                | `128`                                |
| `nacos.resources.requests.cpu`                            | Maximum Request on CPU Resources for Nacos.                                                 | `128`                                |
| `nacos.resources.requests.memory`                         | Maximum Request on Memory Resources for Nacos.                                              | `128`                                |

### Ingress

| Key                                                         | Description                                                                                             | Default                              |
|-------------------------------------------------------------|---------------------------------------------------------------------------------------------------------|--------------------------------------|
| `ingress.enabled`                                           | Enable Ingress Status.                                                                                  | `true`                               |
| `ingress.hosts.admin`                                       |                                                                                                         | `admin.dubbo.domain`                 |
| `ingress.hosts.prometheus`                                  |                                                                                                         | `prom.dubbo.domain`                  |
| `ingress.hosts.grafana`                                     |                                                                                                         | `grafana.dubbo.domain`               |
| `ingress.nameOverride`                                      | Name Override for Ingress.                                                                              | `{}`                                 |
| `ingress.namespaceOverride`                                 | NameSpace Override Ingress.                                                                             | `{}`                                 |
| `ingress.labels`                                            | Labels for  Ingress.                                                                                    | `{}`                                 |       
| `ingress.annotations`                                       | Annotations for Ingress.                                                                                | `{}`                                 |       
| `ingress.nodeSelector`                                      | Node Scheduling for Ingress.                                                                            | `{}`                                 |      
| `ingress.replicas`                                          | Replica Deployment Count for Ingress.                                                                   | `1`                                  |
| `ingress.image.registry`                                    | Image Name for Ingress.                                                                                 | `docker.io/traefik`                  |
| `ingress.image.tag`                                         | Version Tag for Ingress.                                                                                | `v2.10.4`                            |
| `ingress.image.pullPolicy`                                  | Pull Policy for Ingress.                                                                                | `IfNotPresent`                       |
| `ingress.readinessProbe.failureThreshold`                   | The allowed number of failures for the Ingress container.                                               | `1`                                  |
| `ingress.readinessProbe.initialDelaySeconds`                | Initialization Wait Time in Seconds After Ingress Container Starts.                                     | `2`                                  |
| `ingress.readinessProbe.periodSeconds`                      | The Ingress container periodically checks availability.                                                 | `10`                                 |
| `ingress.readinessProbe.successThreshold`                   | The success threshold for the Ingress container.                                                        | `1`                                  |
| `ingress.readinessProbe.timeoutSeconds`                     | Response Timeout Duration in Seconds After Ingress Container Starts.                                    | `2`                                  |
| `ingress.httpGet.path`                                      | Checking the container with an HTTP GET request to a target path.                                       | `/ping`                              |
| `ingress.httpGet.port`                                      | Checking the container with an HTTP GET request to a target port.                                       | `9000`                               |
| `ingress.httpGet.scheme`                                    | Define the network protocol used for health check probe requests when the container starts.             | `HTTP`                               |
| `ingress.livenessProbe.failureThreshold`                    | The allowed number of failures for the Ingress container.                                               | `3`                                  |
| `ingress.livenessProbe.initialDelaySeconds`                 | Initialization Wait Time in Seconds After Ingress Container Starts.                                     | `2`                                  |
| `ingress.livenessProbe.periodSeconds`                       | The Ingress container periodically checks availability.                                                 | `10`                                 |
| `ingress.livenessProbe.successThreshold`                    | The success threshold for the Ingress container.                                                        | `1`                                  |
| `ingress.livenessProbe.timeoutSeconds`                      | Response Timeout Duration in Seconds After Ingress Container Starts.                                    | `2`                                  |
| `ingress.httpGet.path`                                      | Checking the container with an HTTP GET request to a target path.                                       | `/ping`                              |
| `ingress.httpGet.port`                                      | Checking the container with an HTTP GET request to a target port.                                       | `9000`                               |
| `ingress.httpGet.scheme`                                    | Define the network protocol used for health check probe requests when the container starts.             | `HTTP`                               |
| `ingress.strategy.rollingUpdate.maxSurge`                   | Ingress Update Strategy Expected Replicas.                                                              | `1`                                  |
| `ingress.strategy.rollingUpdate.maxUnavailable`             | Ingress Update Strategy Maximum Unavailable Replicas.                                                   | `0`                                  |
| `ingress.securityContext.runAsUser`                         | Non-privileged user identifier for Ingress security context.                                            | `65532`                              |
| `ingress.securityContext.runAsGroup`                        | Non-privileged group identity identifier for Ingress security context.                                  | `65532`                              |
| `ingress.securityContext.runAsNonRoot`                      | Whether the security context for Ingress runs as a non-privileged user.                                 | `true`                               |
| `ingress.containersecurityContext.capabilities.drop`        | Whether Linux kernel capabilities or permissions are enabled in the Ingress container security context. | `[ALL]`                              |
| `ingress.containersecurityContext.readOnlyRootFilesystem`   | Whether the root file system is read-only in the Ingress container security context.                    | `true`                               |
| `ingress.containersecurityContext.allowPrivilegeEscalation` | Whether the Ingress container security context allows privilege escalation.                             | `false`                              |
| `ingress.resources.limits.cpu`                              | Maximum Limit on CPU Resources for Ingress.                                                             | `128`                                |
| `ingress.resources.limits.memory`                           | Maximum Limit on Memory Resources for Ingress.                                                          | `128`                                |
| `ingress.resources.requests.cpu`                            | Maximum Request on CPU Resources for Ingress.    `                                                      | `128`                                |
| `ingress.resources.requests.memory`                         | Maximum Request on Memory Resources for Ingress.                                                        | `128`                                |