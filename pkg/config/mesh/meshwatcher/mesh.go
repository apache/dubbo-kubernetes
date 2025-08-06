package meshwatcher

import (
	"google.golang.org/protobuf/proto"

	meshconfig "istio.io/api/mesh/v1alpha1"
)

// MeshConfigResource holds the current MeshConfig state
type MeshConfigResource struct {
	*meshconfig.MeshConfig
}

func (m MeshConfigResource) ResourceName() string { return "MeshConfigResource" }

func (m MeshConfigResource) Equals(other MeshConfigResource) bool {
	return proto.Equal(m.MeshConfig, other.MeshConfig)
}

// MeshNetworksResource holds the current MeshNetworks state
type MeshNetworksResource struct {
	*meshconfig.MeshNetworks
}

func (m MeshNetworksResource) ResourceName() string { return "MeshNetworksResource" }

func (m MeshNetworksResource) Equals(other MeshNetworksResource) bool {
	return proto.Equal(m.MeshNetworks, other.MeshNetworks)
}
