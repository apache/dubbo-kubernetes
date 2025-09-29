# Istio Service Registry for Dubbo-Kubernetes

This package provides Istio service mesh integration for Dubbo-Kubernetes service discovery.

## Features

- **Service Discovery**: Automatic discovery of services through Istio Pilot
- **Traffic Management**: Support for Istio VirtualServices and DestinationRules
- **Load Balancing**: Leverage Istio's advanced load balancing capabilities  
- **Security**: mTLS support for secure service communication
- **Observability**: Integration with Istio's monitoring and tracing

## Quick Start

### 1. Enable Istio Registry

Add Istio to the list of enabled registries:

```bash
./sail-discovery --registries=Istio
```

### 2. Configure Connection

Set the Istio Pilot address:

```bash
./sail-discovery \
  --registries=Istio \
  --istio-pilot-address=istiod.istio-system.svc.cluster.local:15010 \
  --istio-namespace=istio-system
```

### 3. Deploy in Kubernetes

Use the provided Kubernetes manifests:

```bash
kubectl apply -f examples/istio-service-registry.yaml
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PILOT_ADDRESS` | `istiod.istio-system.svc.cluster.local:15010` | Istio Pilot address |
| `ISTIO_NAMESPACE` | `istio-system` | Istio namespace |
| `TLS_ENABLED` | `true` | Enable TLS connection |
| `SYNC_TIMEOUT` | `30s` | Sync timeout duration |

### Command Line Flags

- `--istio-pilot-address`: Istio Pilot discovery service address
- `--istio-namespace`: Namespace for Istio service discovery
- `--istio-tls-enabled`: Enable TLS for Pilot connection
- `--istio-service-name`: Service name for registration
- `--istio-service-version`: Service version for registration

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│ Dubbo           │    │ Sail Discovery   │    │ Istio Pilot     │
│ Applications    │◄──►│ (Istio Registry) │◄──►│ (xDS Server)    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌──────────────────┐
                       │ Service Model    │
                       │ Conversion       │
                       └──────────────────┘
```

## Components

### Core Components

- **Controller**: Main service registry controller
- **Client**: gRPC client for Istio Pilot communication  
- **Converter**: Service model conversion between Istio and Dubbo
- **Config Manager**: Configuration management and validation

### Data Models

- **IstioConfig**: Configuration for Istio integration
- **IstioServiceInfo**: Service information from Istio
- **VirtualService**: Istio traffic management rules
- **DestinationRule**: Istio load balancing configuration

## Usage Examples

### Basic Configuration

```go
import "github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/istio"

config := istio.DefaultConfig()
config.PilotAddress = "istiod.istio-system.svc.cluster.local:15010"
config.Namespace = "istio-system"

controller := istio.NewController(config, clusterID)
```

### Advanced Configuration

```go
config := &istio.IstioConfig{
    PilotAddress:    "custom-pilot:15010",
    Namespace:       "production",
    TLSEnabled:      true,
    CertPath:        "/etc/certs/cert.pem",
    KeyPath:         "/etc/certs/key.pem",
    CAPath:          "/etc/certs/ca.pem",
    SyncTimeout:     30 * time.Second,
}

controller := istio.NewController(config, clusterID)
```

## Testing

Run the test suite:

```bash
go test ./sail/pkg/serviceregistry/istio/... -v
```

Run benchmarks:

```bash
go test ./sail/pkg/serviceregistry/istio/... -bench=.
```

## Monitoring

### Health Checks

Check controller health:

```go
if controller.IsHealthy() {
    log.Info("Istio service registry is healthy")
}
```

### Metrics

Available metrics:

- Service count: `controller.GetServiceCount()`
- Sync status: `controller.HasSynced()`
- Connection status: `pilotClient.IsConnected()`

## Development

### Building

```bash
go build ./sail/pkg/serviceregistry/istio/...
```

### Code Structure

```
istio/
├── types.go          # Data type definitions
├── client.go         # Istio Pilot client
├── config.go         # Configuration management
├── converter.go      # Service model conversion
├── controller.go     # Main service registry controller
└── istio_test.go     # Test cases
```

### Adding Features

1. Define new types in `types.go`
2. Implement functionality in appropriate files
3. Add comprehensive tests
4. Update documentation

## Compatibility

- **Istio**: 1.15+ (tested with 1.15, 1.16, 1.17)
- **Kubernetes**: 1.23+
- **Go**: 1.19+

## Troubleshooting

### Common Issues

1. **Connection failures**: Check Istio Pilot address and network connectivity
2. **TLS errors**: Verify certificate paths and validity
3. **Service not found**: Check namespace configuration
4. **Sync issues**: Review timeout settings and resource limits

### Debug Logging

Enable debug logging:

```bash
--log-level=debug
```

### Support

- [Documentation](../../docs/istio-service-registry.md)
- [Examples](../../examples/)
- [GitHub Issues](https://github.com/apache/dubbo-kubernetes/issues)

## Contributing

We welcome contributions! Please see:

- [Contributing Guide](../../CONTRIBUTING.md)
- [Code of Conduct](../../CODE_OF_CONDUCT.md)
- [Development Setup](../../docs/development.md)

## License

This project is licensed under the Apache License 2.0. See [LICENSE](../../LICENSE) for details.