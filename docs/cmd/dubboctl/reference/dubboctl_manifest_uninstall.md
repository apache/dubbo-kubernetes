## dubboctl manifest uninstall

uninstall dubbo control plane

### Synopsis

Uninstall the specified component. Unconditional uninstallation is currently not supported (this means that users cannot
force deletion if they do not know the DubboConfig yaml or set parameters used in dubboctl manifest intall, and need to
use kubectl and other tools to delete it).
Typical use cases are:

```sh
dubboctl manifest uninstall
```

| parameter     | shorthand | describe                                                                                                                                                                    | Example                                                                                         | required |
|---------------|-----------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------|----------|
| --filenames   | -f        | Specify one or more user-defined DubboConfig yaml paths, and overlay them in order from left to right when parsing.                                                         | dubboctl manifest install -f path/to/file0.yaml, path/to/file1.yaml                             | No       |
| --charts      |           | The directory where Helm Charts are stored. If the user does not specify it, /deploy/charts is used by default.                                                             | dubboctl manifest install --charts path/to/charts                                               | No       |
| --profiles    |           | The directory where profiles are stored. If the user does not specify it, /deploy/profiles is used by default.                                                              | dubboctl manifest install --profiles path/to/profiles                                           | No       |
| --set         | -s        | Set one or more key-value pairs in DubboConfig yaml. The priority is set flags > profile > user-defined DubboOperator yaml. It is recommended not to use set in production. | dubboctl manifest install --set components.admin.replicas=2,components.admin.rbac.enabled=false | 否        |
| --ku beConfig |           | The path to store kubeconfig                                                                                                                                                | dubboctl manifest install --kubeConfig path/to/kubeConfig                                       | No       |
| --context     |           | Specify to use the context in kubeconfig                                                                                                                                    | dubboctl manifest install --context contextVal                                                  | No       |

### SEE ALSO

* [dubboctl manifest](dubboctl_manifest.md) - Commands help user to generate manifest and install manifest
