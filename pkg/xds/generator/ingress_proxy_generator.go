package generator

import (
	"context"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	model "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	xds_context "github.com/apache/dubbo-kubernetes/pkg/xds/context"
)

var ingressLog = core.Log.WithName("ingress-proxy-generator")

// Ingress is a marker to indicate by which ProxyGenerator resources were generated.
const Ingress = "outbound"

type IngressGenerator struct{}

func (g IngressGenerator) Generator(ctx context.Context, _ *model.ResourceSet, xdsCtx xds_context.Context, proxy *model.Proxy) (*model.ResourceSet, error) {
	return nil, nil
}
