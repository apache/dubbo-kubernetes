package serviceregistry

import (
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/provider"
)

// Instance of a service registry. A single service registry combines the capabilities of service discovery
// and the controller for managing asynchronous events.
type Instance interface {
	model.Controller
	model.ServiceDiscovery

	// Provider backing this service registry (i.e. Kubernetes etc.)
	Provider() provider.ID

	// Cluster for which the service registry applies. Only needed for multicluster systems.
	Cluster() cluster.ID
}
