package aggregate

import (
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry"
	"k8s.io/klog/v2"
	"sync"
)

type Controller struct {
	registries      []*registryEntry
	storeLock       sync.RWMutex
	running         bool
	meshHolder      mesh.Holder
	configClusterID cluster.ID
}

type registryEntry struct {
	serviceregistry.Instance
	// stop if not nil is the per-registry stop chan. If null, the server stop chan should be used to Run the registry.
	stop <-chan struct{}
}

type Options struct {
	MeshHolder      mesh.Holder
	ConfigClusterID cluster.ID
}

func NewController(opt Options) *Controller {
	return &Controller{
		registries:      make([]*registryEntry, 0),
		configClusterID: opt.ConfigClusterID,
		meshHolder:      opt.MeshHolder,
		running:         false,
	}
}

// Run starts all the controllers
func (c *Controller) Run(stop <-chan struct{}) {
	c.storeLock.Lock()
	for _, r := range c.registries {
		// prefer the per-registry stop channel
		registryStop := stop
		if s := r.stop; s != nil {
			registryStop = s
		}
		go r.Run(registryStop)
	}
	c.running = true
	c.storeLock.Unlock()

	<-stop
	klog.Info("Registry Aggregator terminated")
}
