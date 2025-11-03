package model

const (
	APITypePrefix              = "type.googleapis.com/"
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
