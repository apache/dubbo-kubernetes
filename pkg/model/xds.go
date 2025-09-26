package model

const (
	APITypePrefix              = "type.googleapis.com/"
	ClusterType                = APITypePrefix + "envoy.config.cluster.v3.Cluster"
	ListenerType               = APITypePrefix + "envoy.config.listener.v3.Listener"
	EndpointType               = APITypePrefix + "envoy.config.endpoint.v3.ClusterLoadAssignment"
	RouteType                  = APITypePrefix + "envoy.config.route.v3.RouteConfiguration"
	SecretType                 = APITypePrefix + "envoy.extensions.transport_sockets.tls.v3.Secret"
	ExtensionConfigurationType = APITypePrefix + "envoy.config.core.v3.TypedExtensionConfig"

	HealthInfoType = APITypePrefix + "dubbo.v1.HealthInformation"
	DebugType      = "dubbo.io/debug"
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
	default:
		return typeURL
	}
}
