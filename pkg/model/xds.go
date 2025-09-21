package model

const (
	APITypePrefix = "type.googleapis.com/"
	ClusterType   = APITypePrefix + "envoy.config.cluster.v3.Cluster"
	ListenerType  = APITypePrefix + "envoy.config.listener.v3.Listener"
	EndpointType  = APITypePrefix + "envoy.config.endpoint.v3.ClusterLoadAssignment"
	RouteType     = APITypePrefix + "envoy.config.route.v3.RouteConfiguration"

	DebugType = "dubbo.io/debug"
)
