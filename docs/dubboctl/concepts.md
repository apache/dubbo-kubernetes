# Concepts

## dubboctl Manifest and Profile

### Manifest

Dubbo-kubernetes, as a central control component, needs to launch various components such as Grafana, Prometheus,
Skywalking, Zipkin, Nacos, Zookeeper, etc., to provide additional functionality. In a Kubernetes environment, the
configuration YAML that describes these components is referred to as the "manifest." It's also the final result
processed by dubboctl, which can be used with tools like kubectl to directly launch the required components.

Dubboctl utilizes the Helm API to render the Helm Charts for various components to generate the manifest. Among these,
the Helm Charts for Admin and Nacos are maintained by the Dubbo-admin community, while other mature components use Helm
Charts provided by their official sources. (For more information on how Helm works, please refer
to https://helm.sh/zh/docs/)

### DubboConfig YAML (Profile)

DubboConfig YAML and Profile essentially have the same format and field meanings in YAML configuration files. However,
they serve different purposes and are kept separate. DubboConfig YAML is user-facing and represents custom requirements,
whereas profiles are refined and provided by Dubbo-admin (users can also set custom profiles) to represent basic
configurations for different scenarios. Currently, the community offers two types of profiles: "default" and "demo."

A typical DubboConfig YAML looks like this:

```yaml
# DubboConfig example
apiVersion: dubbo.apache.org/v1alpha1
kind: DubboConfig
metadata:
  namespace: dubbo-system
spec:
  # kubectl basic metadata

  # Profile specifies the base default configuration to use. If using the profile provided by Dubbo-admin,
  # you can currently specify either "default" or "demo."
  profile: default
  namespace: dubbo-system

  # Metadata for configuring components. For components maintained by Dubbo-admin, you can configure "enabled,"
  # while other mature components can have repository URLs and Helm Chart versions.
  componentsMeta:
    # Components maintained by Dubbo-admin
    admin:
      enabled: true
    # Mature components
    grafana:
      enabled: true
      repoURL: https://grafana.github.io/helm-charts
      version: 6.52.4

  # Configure each component, where the values in components.admin and components.grafana fields
  # will be mapped directly to the values.yaml of the respective Helm Chart.
  components:
    admin:
      replicas: 2
    grafana:
      testFramework:
        enabled: true
```

The default profile is as follows:

```yaml
apiVersion: dubbo.apache.org/v1alpha1
kind: DubboOperator
metadata:
  namespace: dubbo-system
spec:
  profile: default
  namespace: dubbo-system
  # The default profile launches admin and zookeeper components by default
  componentsMeta:
    admin:
      enabled: true
    zookeeper:
      enabled: true
      repoURL: https://charts.bitnami.com/bitnami
      version: 11.1.6
```

The demo profile is as follows:

```yaml
apiVersion: dubbo.apache.org/v1alpha1
kind: DubboOperator
metadata:
  namespace: dubbo-system
spec:
  profile: demo
  namespace: dubbo-system
  # The demo profile launches all components by default
  componentsMeta:
    admin:
      enabled: true
    grafana:
      enabled: true
      repoURL: https://grafana.github.io/helm-charts
      version: 6.52.4
    nacos:
      enabled: true
    zookeeper:
      enabled: true
      repoURL: https://charts.bitnami.com/bitnami
      version: 11.1.6
    prometheus:
      enabled: true
      repoURL: https://prometheus-community.github.io/helm-charts
      version: 20.0.2
    skywalking:
      enabled: true
      repoURL: https://apache.jfrog.io/artifactory/skywalking-helm
      version: 4.3.0
    zipkin:
      enabled: true
      repoURL: https://openzipkin.github.io/zipkin
      version: 0.3.0
```

You can see that DubboConfig consists of three parts: dubboctl metadata, componentsMeta, and components. The dubboctl
metadata can specify the profile, componentsMeta controls the toggles for components, and mature components can have
repository addresses and Helm Chart versions specified. Here, let's focus on components. As mentioned in the manifest
section, dubboctl uses the Helm API to render Helm Charts, and the values.yaml required by each chart is derived from
the field values in the components section. For example, the values in components.admin will be directly mapped to the
values.yaml of the Dubbo-kubernetes
chart (https://github.com/apache/dubbo-kubernetes/blob/master/deploy/charts/dubbo-admin/values.yaml).

### Overlay

Profiles, as basic configuration items, can be applied over user-defined DubboConfig YAML using an Overlay mechanism.
For example, overlaying the DubboConfig YAML example from the previous section onto the default profile results in:

```yaml
# DubboConfig example
apiVersion: dubbo.apache.org/v1alpha1
kind: DubboConfig
metadata:
  namespace: dubbo-system
spec:
  # kubectl basic metadata

  # Profile specifies the base default configuration to use. If using the profile provided by Dubbo-admin,
  # you can currently specify either "default" or "demo."
  profile: default
  namespace: dubbo-system

  # Metadata for configuring components. For components maintained by Dubbo-admin, you can configure "enabled,"
  # while other mature components can have repository URLs and Helm Chart versions.
  componentsMeta:
    # Components maintained by Dubbo-admin
    admin:
      enabled: true
    # Mature components
    grafana:
      enabled: true
      repoURL: https://grafana.github.io/helm-charts
      version: 6.52.4
    zookeeper:
      enabled: true
      repoURL: https://charts.bitnami.com/bitnami
      version: 11.1.6

  # Configure each component, where the values in components.admin and components.grafana fields
  # will be mapped directly to the values.yaml of the respective Helm Chart.
  components:
    admin:
      replicas: 2
    grafana:
      testFramework:
        enabled: true
```

Currently, Overlay is implemented using JSON Merge Patch (https://datatracker.ietf.org/doc/html/rfc7396).

## dubboctl Create

### .dubbo Directory

When you create an application using dubboctl in the correct manner, you'll notice that a ".dubbo" directory is
generated. The purpose of this hidden file is to determine whether the application needs to build an image. If the code
hasn't been changed, it won't build an image by default unless you include the "--force" flag.

## dubbo.yaml

dubbo.yaml is a metadata recording file that contains a portion of the data used by dubboctl when it runs. Some of your
actions will be recorded in dubbo.yaml, and it takes precedence when you use