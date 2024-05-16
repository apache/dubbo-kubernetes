## dubboctl manifest diff generate

Generate the manifest of the required components.

### Synopsis

Generate the manifest of the required components.
Typical use cases are:

```sh
dubboctl manifest generate | kubectl apply -f -
```

| parameter    | shorthand | describe                                                                                                                                                   | Example                                                                                                         | required |
|--------------|-----------|------------------------------------------------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------------------------------|----------|
| --filenames  | -f        | Specify one or more user-defined DubboConfig yaml paths. When parsing, follow the order from left to right. Overlay                                        | dubboctl manifest generate -f path/to/file0.yaml, path/to/file1.yaml                                            | No       |
| --charts     |           | The directory where Helm Charts are stored. If the user does not specify it, /deploy/charts is used by default                                             | dubboctl manifest generate --charts path/to/charts                                                              | No       |
| --profiles   |           | The directory where profiles are stored. If the user does not specify it, /deploy/profiles is used by default                                              | dubboctl manifest generate --profiles path/to/profiles                                                          | No       |
| --set        | -s        | Set one or more key-value pairs in DubboConfig yaml. The priority is set flags > user-defined DubboConfig yaml > profile. It is recommended not to use set | dubboctl manifest generate --set components.admin.replicas=2,components in production. admin.rbac.enabled=false | No       |
| --kubeConfig |           | The path where kubeconfig is stored                                                                                                                        | dubboctl manifest generate --kubeConfig path/to/kubeConfig                                                      | No       |
| --context    |           | Specify the context in kubeconfig                                                                                                                          | dubboctl manifest generate --context contextVal                                                                 | No       |
| --output     | -o        | Specify the output path for the final generated manifest. If not set, the output will be output to the console by default                                  | dubboctl manifest generate -o path/to/target/directory                                                          | No       |

### SEE ALSO

* [dubboctl manifest](dubboctl_manifest.md) - Commands help user to generate manifest and install manifest
