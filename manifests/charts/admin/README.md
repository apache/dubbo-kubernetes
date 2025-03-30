# Admin         

![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)
![Helm: v3](https://img.shields.io/static/v1?label=Helm&message=v3&color=informational&logo=helm)

| Key                                              | Description                                                                                | Default                                   |
|--------------------------------------------------|--------------------------------------------------------------------------------------------|-------------------------------------------|
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

