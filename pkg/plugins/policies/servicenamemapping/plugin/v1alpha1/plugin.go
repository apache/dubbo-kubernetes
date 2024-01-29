package v1alpha1

import (
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_plugins "github.com/apache/dubbo-kubernetes/pkg/core/plugins"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/policies/core/matchers"
	api "github.com/apache/dubbo-kubernetes/pkg/plugins/policies/servicenamemapping/api/v1alpha1"
	xds_context "github.com/apache/dubbo-kubernetes/pkg/xds/context"
)

var _ core_plugins.PolicyPlugin = &plugin{}
var log = core.Log.WithName("ServiceNameMapping")

type plugin struct {
}

func NewPlugin() core_plugins.Plugin {
	return &plugin{}
}

func (p plugin) MatchedPolicies(dataplane *core_mesh.DataplaneResource, resources xds_context.Resources) (core_xds.TypedMatchingPolicies, error) {
	return matchers.MatchedPolicies(api.ServiceNameMappingType, dataplane, resources)
}

func (p plugin) Apply(rs *core_xds.ResourceSet, ctx xds_context.Context, proxy *core_xds.Proxy) error {
	log.Info("apply is not implemented")
	return nil
}
