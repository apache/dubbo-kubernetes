## dubboctl manifest

Commands help user to generate manifest and install manifest

### Synopsis

Commands help user to generate manifest and install manifest

```
    -h, --help help for config
```

### SEE ALSO

* [dubboctl](dubboctl.md) - Management tool for dubbo-kubernetes
* [dubboctl manifest diff](dubboctl_manifest_diff.md) - show the difference between two files or dirs
* [dubboctl manifest generate](dubboctl_manifest_generate.md) - Generate the manifest of the required components.
* [dubboctl manifest install](dubboctl_manifest_install.md) - Install the required components directly to the k8s
  cluster.
* [dubboctl manifest uninstall](dubboctl_manifest_uninstall.md) - Uninstall the specified component. Unconditional
  uninstallation is currently not supported (this means that users cannot force deletion if they do not know the
  DubboConfig yaml or set parameters used in dubboctl manifest intall, and need to use kubectl and other tools to delete
  it).