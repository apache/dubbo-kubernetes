# How to Customize and Build a Template

You can refer to this project: https://github.com/sjmshsh/dubboctl-samples

Here's the directory structure for `dubboctl-samples`:

```
shCopy code.
├── go
│   ├── grpc
│   │   ├── api
│   │   │   ├── generate.sh
│   │   │   ├── samples_api.pb.go
│   │   │   ├── samples_api.proto
│   │   │   └── samples_api_triple.pb.go
│   │   ├── cmd
│   │   │   └── client.go
│   │   ├── conf
│   │   │   └── dubbogo.yml
│   │   └── go.mod
│   └── mesh
│       ├── api
│       │   ├── generate.sh
│       │   ├── samples_api.pb.go
│       │   ├── samples_api.proto
│       │   └── samples_api_triple.pb.go
│       ├── cmd
│       │   └── client.go
│       ├── conf
│       │   └── dubbogo.yml
│       └── go.mod
└── java
    ├── metrics
    │   ├── case-configuration.yml
    │   ├── case-versions.conf
    │   ├── pom.xml
    │   ├── README.md
    │   └── src
    │       ├── main
    │       │   ├── java
    │       │   │   └── org
    │       │   │       └── apache
    │       │   │           └── dubbo
    │       │   │               └── samples
    │       │   │                   └── metrics
    │       │   │                       ├── api
    │       │   │                       │   └── DemoService.java
    │       │   │                       ├── EmbeddedZooKeeper.java
    │       │   │                       ├── impl
    │       │   │                       │   └── DemoServiceImpl.java
    │       │   │                       ├── MetricsConsumer.java
    │       │   │                       ├── MetricsProvider.java
    │       │   │                       └── model
    │       │   │                           ├── Result.java
    │       │   │                           └── User.java
    │       │   └── resources
    │       │       ├── log4j.properties
    │       │       └── spring
    │       │           ├── dubbo-demo-consumer.xml
    │       │           └── dubbo-demo-provider.xml
    │       └── test
    │           └── java
    │               └── org
    │                   └── apache
    │                       └── dubbo
    │                           └── samples
    │                               └── metrics
    │                                   └── MetricsServiceIT.java
    └── spring-xml
        ├── case-configuration.yml
        ├── case-versions.conf
        ├── pom.xml
        └── src
            ├── main
            │   ├── java
            │   │   └── org
            │   │       └── apache
            │   │           └── dubbo
            │   │               └── samples
            │   │                   ├── api
            │   │                   │   └── GreetingsService.java
            │   │                   ├── client
            │   │                   │   ├── AlwaysApplication.java
            │   │                   │   └── Application.java
            │   │                   └── provider
            │   │                       ├── Application.java
            │   │                       └── GreetingsServiceImpl.java
            │   └── resources
            │       ├── log4j.properties
            │       └── spring
            │           ├── dubbo-demo-consumer.xml
            │           └── dubbo-demo-provider.xml
            └── test
                └── java
                    └── org
                        └── apache
                            └── dubbo
                                └── samples
                                    └── test
                                        └── GreetingsServiceIT.java
```

In a nutshell:

```
shCopy code.
├── go
│   ├── grpc
│   └── mesh
└── java
    ├── metrics
    └── spring-xml
```

"go" and "java" represent your runtime language. "grpc," "mesh," "metrics," and "spring-xml" are the application names.
The specific template code should be written in the "grpc," "mesh," "metrics," or "spring-xml" directories.

Now that you've customized and built your template, you can proceed to read the instructions in the dubboctl repository
on how to add your custom-built template to the repository.

This guide should help you understand the structure and process in a straightforward manner.