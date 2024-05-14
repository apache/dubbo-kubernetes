## Use of environment variables in projects

Among the common commands create, repository, build and deploy. We will see that there are many flags. For example, the
deploy flags are as follows:

```sh
   -a, --apply Whether to apply the application to the k8s cluster by the way
   -b, --builder-image string Specify a custom builder image for use by the builder other than its default.
       --containerPort int The port of the deployment to listen on pod (required)
   -e, --envs stringArray Environment variable to set in the form NAME=VALUE. This is for the environment variables passed in by the builderpack build method.
   -f, --force Whether to force push
   -h, --help help for deploy
   -i, --image string Container image( [registry]/[namespace]/[name]:[tag] )
       --imagePullPolicy string The image pull policy of the deployment, default to IfNotPresent
       --limitCpu int The limit cpu to deploy (default 1000)
       --limitMem int The limit memory to deploy (default 1024)
       --maxReplicas int The max replicas to deploy (default 10)
       --minReplicas int The min replicas to deploy (default 3)
       --name string The name of deployment (required)
   -n, --namespace string Deploy into a specific namespace (default "dubbo-system")
       --nobuild Skip the step of building the image.
       --nodePort int The nodePort of the deployment to expose
   -o, --output string output kubernetes manifest (default "kube.yaml")
   -p, --path string Path to the application. Default is current directory ($DUBBO_PATH)
       --push Whether to push the image to the registry center by the way
       --replicas int The number of replicas to deploy (default 3)
       --requestCpu int The request cpu to deploy (default 500)
       --requestMem int The request memory to deploy (default 512)
       --revisions int The number of replicas to deploy (default 5)
       --secret string The secret to image pull from registry
       --serviceAccount string TheServiceAccount for the deployment
       --targetPort int The targetPort of the deployment, default to port
   -d, --useDockerfile Use the dockerfile with the specified path to build
```

We can add it to the environment variable with the prefix of `DUBBO_` and the non-abbreviated notation of flags in all
capital letters. Subsequent operations will give priority to the value in the environment variable. The priority order
from large to small is: environment variables, command line flags, dubbo.yaml.

As an example:

```sh
export DUBBO_PUSH=true
```

Add the --push flag to the environment variables

Therefore, future operations PUSH will always be true unless the setting of the environment variable is cancelled.