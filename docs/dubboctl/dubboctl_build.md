## dubboctl build

Build an application container locally without deploying

### Synopsis

    Usage:
    dubboctl build [flags]

    Flags:
    -b, --builder-image string Specify a custom builder image for use by the builder other than its default.
    -e, --envs stringArray environment variable for an application, KEY=VALUE format
    -f, --force Whether to force push
    -h, --help help for build
    -i, --image string Container image( [registry]/[namespace]/[name]:[tag] )
    -p, --path string Path to the application. Default is current directory ($DUBBO_PATH)
    --push Whether to push the image to the registry center by the way
    -d, --useDockerfile Use the dockerfile with the specified path to build

### SEE ALSO

* [dubboctl](reference/dubboctl.md) - Management tool for dubbo-kubernetes