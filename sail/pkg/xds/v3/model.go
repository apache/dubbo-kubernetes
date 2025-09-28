package v3

import "github.com/apache/dubbo-kubernetes/pkg/model"

const (
	ClusterType    = model.ClusterType
	ListenerType   = model.ListenerType
	EndpointType   = model.EndpointType
	RouteType      = model.RouteType
	DebugType      = model.DebugType
	HealthInfoType = model.HealthInfoType
)

func GetShortType(typeURL string) string {
	return model.GetShortType(typeURL)
}
