package generator

import (
	"context"
	"fmt"
	model "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/policies/core/generator"
	xds_context "github.com/apache/dubbo-kubernetes/pkg/xds/context"
	"github.com/apache/dubbo-kubernetes/pkg/xds/generator/core"
)

const (
	DefaultProxy = "default-proxy"
	IngressProxy = "ingress-proxy"
)

type ProxyTemplateGenerator struct {
	ProfileName []string
}

func (g *ProxyTemplateGenerator) Generate(ctx context.Context, xdsCtx xds_context.Context, proxy *model.Proxy) (*model.ResourceSet, error) {
	resources := model.NewResourceSet()

	for _, name := range g.ProfileName {
		p, ok := predefinedProfiles[name]
		if !ok {
			return nil, fmt.Errorf("profile{name=%q}: unknown profile", g.ProfileName)
		}
		if rs, err := p.Generator(ctx, resources, xdsCtx, proxy); err != nil {
			return nil, err
		} else {
			resources.AddSet(rs)
		}
	}

	return resources, nil
}

func NewDefaultProxyProfile() core.ResourceGenerator {
	return core.CompositeResourceGenerator{
		InboundProxyGenerator{},
		OutboundProxyGenerator{},
		generator.NewGenerator(),
	}
}

var predefinedProfiles = make(map[string]core.ResourceGenerator)

func init() {
	RegisterProfile(DefaultProxy, NewDefaultProxyProfile())
	RegisterProfile(IngressProxy, core.CompositeResourceGenerator{IngressGenerator{}})
}

func RegisterProfile(profileName string, generator core.ResourceGenerator) {
	predefinedProfiles[profileName] = generator
}
