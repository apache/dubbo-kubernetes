package servicenamemapping

import (
	"github.com/apache/dubbo-kubernetes/pkg/plugins/policies/core"
	api_v1alpha1 "github.com/apache/dubbo-kubernetes/pkg/plugins/policies/servicenamemapping/api/v1alpha1"
	k8s_v1alpha1 "github.com/apache/dubbo-kubernetes/pkg/plugins/policies/servicenamemapping/k8s/v1alpha1"
	plugin_v1alpha1 "github.com/apache/dubbo-kubernetes/pkg/plugins/policies/servicenamemapping/plugin/v1alpha1"
)

func init() {
	core.Register(
		api_v1alpha1.ServiceNameMappingResourceTypeDescriptor,
		k8s_v1alpha1.AddToScheme,
		plugin_v1alpha1.NewPlugin(),
	)
}
