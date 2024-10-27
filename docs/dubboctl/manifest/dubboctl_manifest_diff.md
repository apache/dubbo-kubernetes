## dubboctl manifest diff

show the difference between two files or dirs

### Synopsis

Show the differences between the two manifests, split the manifest into multiple k8s objects, compare objects with the
same namespace:kind:name, and output the redundant objects and objects with parsing errors in the manifest; if you show
the differences between the two directories , perform the above processing on manifests with the same name, and output
the redundant manifests in the directory and the manifests with parsing errors.

Typical use cases are:

```sh
dubboctl manifest diff path/to/file0 path/to/file1
```

| parameter    | shorthand | describe                             | Example                                                       | required |
|--------------|-----------|--------------------------------------|---------------------------------------------------------------|----------|
| --compareDir |           | Compare manifests in two directories | dubboctl manifest diff path/to/dir0 path/to/dir1 --compareDir | No       |

### SEE ALSO

* [dubboctl manifest](dubboctl_manifest.md) - Commands help user to generate manifest and install manifest
