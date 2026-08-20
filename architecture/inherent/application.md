# Inherent Application

Inherent is proxyless. The injector adds the xDS bootstrap, runtime configuration,
certificate volume and telemetry settings to the original application container.
The application consumes them natively; no proxy image, sidecar container, Service
port rewrite or node-level traffic interception is added.
