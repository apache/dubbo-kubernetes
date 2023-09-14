# Quick start

## Install components

Follow the components required by the dubbo project, such as admin and zookeeper.

```sh
dubboctl manifest install
```

## Initialize dubbo project

```sh
dubboctl create -l java
```

Initialize a java project in the current directory using the default template provided by dubboctl. Please make sure the current directory is an empty directory.

## Deploy to k8s

```sh
dubboctl deploy --containerPort 20000 --push --image docker.io/testuser/testdubbo:latest --apply
```



> If you do not plan to deploy the application to k8s, please use the build command to build the image and replace the third step with
>
> ```sh
> dubboctl build --push --image docker.io/testuser/testdubbo:latest
> ```