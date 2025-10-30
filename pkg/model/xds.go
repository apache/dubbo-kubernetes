package model

import (
	"strings"
)

const (
	APITypePrefix   = "type.googleapis.com/"
	envoyTypePrefix = APITypePrefix + "envoy."

	ClusterType                = APITypePrefix + "envoy.config.cluster.v3.Cluster"
	EndpointType               = APITypePrefix + "envoy.config.endpoint.v3.ClusterLoadAssignment"
	ListenerType               = APITypePrefix + "envoy.config.listener.v3.Listener"
	RouteType                  = APITypePrefix + "envoy.config.route.v3.RouteConfiguration"
	SecretType                 = APITypePrefix + "envoy.extensions.transport_sockets.tls.v3.Secret"
	ExtensionConfigurationType = APITypePrefix + "envoy.config.core.v3.TypedExtensionConfig"

	NameTableType   = APITypePrefix + "dubbo.networking.nds.v1.NameTable"
	HealthInfoType  = APITypePrefix + "dubbo.v1.HealthInformation"
	ProxyConfigType = APITypePrefix + "dubbo.mesh.v1alpha1.ProxyConfig"
	DebugType       = "dubbo.io/debug"
	BootstrapType   = APITypePrefix + "envoy.config.bootstrap.v3.Bootstrap"
	AddressType     = APITypePrefix + "dubbo.workload.Address"
	WorkloadType    = APITypePrefix + "dubbo.workload.Workload"
)

func GetShortType(typeURL string) string {
	switch typeURL {
	case ClusterType:
		return "CDS"
	case ListenerType:
		return "LDS"
	case RouteType:
		return "RDS"
	case EndpointType:
		return "EDS"
	case SecretType:
		return "SDS"
	case NameTableType:
		return "NDS"
	case ProxyConfigType:
		return "PCDS"
	case ExtensionConfigurationType:
		return "ECDS"
	case AddressType, WorkloadType:
		return "WDS"
	default:
		return typeURL
	}
}

func GetMetricType(typeURL string) string {
	switch typeURL {
	case ClusterType:
		return "cds"
	case ListenerType:
		return "lds"
	case RouteType:
		return "rds"
	case EndpointType:
		return "eds"
	case SecretType:
		return "sds"
	case NameTableType:
		return "nds"
	case ProxyConfigType:
		return "pcds"
	case ExtensionConfigurationType:
		return "ecds"
	case BootstrapType:
		return "bds"
	case AddressType, WorkloadType:
		return "wds"
	default:
		return typeURL
	}
}

func GetResourceType(shortType string) string {
	upper := strings.ToUpper(shortType)
	switch upper {
	case "CDS":
		return ClusterType
	case "LDS":
		return ListenerType
	case "RDS":
		return RouteType
	case "EDS":
		return EndpointType
	case "SDS":
		return SecretType
	case "NDS":
		return NameTableType
	case "PCDS":
		return ProxyConfigType
	case "ECDS":
		return ExtensionConfigurationType
	case "WDS":
		return AddressType
	default:
		return shortType
	}
}

func IsEnvoyType(typeURL string) bool {
	return strings.HasPrefix(typeURL, envoyTypePrefix)
}
