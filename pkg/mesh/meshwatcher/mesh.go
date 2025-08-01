package meshwatcher

import meshconfig "istio.io/api/mesh/v1alpha1"

type MeshConfigResource struct {
	*meshconfig.MeshConfig
}
