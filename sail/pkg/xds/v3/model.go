package v3

import "github.com/apache/dubbo-kubernetes/pkg/model"

const (
	ClusterType                = model.ClusterType
	ListenerType               = model.ListenerType
	EndpointType               = model.EndpointType
	RouteType                  = model.RouteType
	SecretType                 = model.SecretType
	ExtensionConfigurationType = model.ExtensionConfigurationType
	NameTableType              = model.NameTableType
	DebugType                  = model.DebugType
	HealthInfoType             = model.HealthInfoType
	AddressType                = model.AddressType
	WorkloadType               = model.WorkloadType
)

func GetShortType(typeURL string) string {
	return model.GetShortType(typeURL)
}
