package secrets

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"

	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	core_ca "github.com/apache/dubbo-kubernetes/pkg/core/ca"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
)

type Identity struct {
	Mesh     string
	Name     string
	Services mesh_proto.MultiValueTagSet
}

type IdentityProvider interface {
	// Get returns PEM encoded cert + key, backend that was used to generate this pair and an error.
	Get(context.Context, Identity, *core_mesh.MeshResource) (*core_xds.IdentitySecret, string, error)
}

type identityCertProvider struct {
	caManagers     core_ca.Managers
	latencyMetrics *prometheus.SummaryVec
}
