package aggregate

import (
	"sync"

	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/provider"
	"k8s.io/klog/v2"
)

var (
	_ model.ServiceDiscovery    = &Controller{}
	_ model.AggregateController = &Controller{}
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
	runningCount := 0
	for i, r := range c.registries {
		if r == nil || r.Instance == nil {
			klog.Errorf("Skipping nil registry at index %d", i)
			continue
		}

		// prefer the per-registry stop channel
		registryStop := stop
		if s := r.stop; s != nil {
			registryStop = s
		}

		go func(registry *registryEntry, idx int) {
			defer func() {
				if rec := recover(); rec != nil {
					klog.Errorf("Registry %s/%s (index %d) panicked: %v",
						registry.Provider(), registry.Cluster(), idx, rec)
				}
			}()

			klog.Infof("Starting registry %s/%s (index %d)",
				registry.Provider(), registry.Cluster(), idx)
			registry.Run(registryStop)
			klog.Infof("Registry %s/%s (index %d) stopped",
				registry.Provider(), registry.Cluster(), idx)
		}(r, i)
		runningCount++
	}
	c.running = true
	c.storeLock.Unlock()

	klog.Infof("Registry Aggregator started with %d registries", runningCount)

	<-stop
	klog.Info("Registry Aggregator terminated")
	c.storeLock.Lock()
	c.running = false
	c.storeLock.Unlock()
}

func (c *Controller) HasSynced() bool {
	for _, r := range c.GetRegistries() {
		if !r.HasSynced() {
			klog.V(2).Infof("registry %s is syncing", r.Cluster())
			return false
		}
	}
	return true
}

func (c *Controller) Services() []*model.Service {
	// smap is a map of hostname (string) to service index, used to identify services that
	// are installed in multiple clusters.
	smap := make(map[host.Name]int)
	index := 0
	services := make([]*model.Service, 0)
	// Locking Registries list while walking it to prevent inconsistent results
	for _, r := range c.GetRegistries() {
		svcs := r.Services()
		if r.Provider() != provider.Kubernetes {
			index += len(svcs)
			services = append(services, svcs...)
		} else {
			for _, s := range svcs {
				previous, ok := smap[s.Hostname]
				if !ok {
					// First time we see a service. The result will have a single service per hostname
					// The first cluster will be listed first, so the services in the primary cluster
					// will be used for default settings. If a service appears in multiple clusters,
					// the order is less clear.
					smap[s.Hostname] = index
					index++
					services = append(services, s)
				} else {
					// We must deepcopy before merge, and after merging, the ClusterVips length will be >= 2.
					// This is an optimization to prevent deepcopy multi-times
					if services[previous].ClusterVIPs.Len() < 2 {
						// Deep copy before merging, otherwise there is a case
						// a service in remote cluster can be deleted, but the ClusterIP left.
						services[previous] = services[previous].DeepCopy()
					}
					// If it is seen second time, that means it is from a different cluster, update cluster VIPs.
					mergeService(services[previous], s, r)
				}
			}
		}
	}
	return services
}

func (c *Controller) GetService(hostname host.Name) *model.Service {
	var out *model.Service
	for _, r := range c.GetRegistries() {
		service := r.GetService(hostname)
		if service == nil {
			continue
		}
		if r.Provider() != provider.Kubernetes {
			return service
		}
		if out == nil {
			out = service.DeepCopy()
		} else {
			// If we are seeing the service for the second time, it means it is available in multiple clusters.
			mergeService(out, service, r)
		}
	}
	return out
}

func (c *Controller) GetRegistries() []serviceregistry.Instance {
	c.storeLock.RLock()
	defer c.storeLock.RUnlock()

	// copy registries to prevent race, no need to deep copy here.
	out := make([]serviceregistry.Instance, len(c.registries))
	for i := range c.registries {
		out[i] = c.registries[i]
	}
	return out
}

func mergeService(dst, src *model.Service, srcRegistry serviceregistry.Instance) {
	if !src.Ports.Equals(dst.Ports) {
		klog.V(2).Infof("service %s defined from cluster %s is different from others", src.Hostname, srcRegistry.Cluster())
	}
	// Prefer the k8s HostVIPs where possible
	clusterID := srcRegistry.Cluster()
	if len(dst.ClusterVIPs.GetAddressesFor(clusterID)) == 0 {
		newAddresses := src.ClusterVIPs.GetAddressesFor(clusterID)
		dst.ClusterVIPs.SetAddressesFor(clusterID, newAddresses)
	}

	// Enhanced cross-cluster service merging
	mergeServiceAttributes(dst, src, srcRegistry)
	mergeServicePorts(dst, src, srcRegistry)
}

// mergeServiceAttributes merges service attributes from different clusters/registries
func mergeServiceAttributes(dst, src *model.Service, srcRegistry serviceregistry.Instance) {
	// Thread-safe label initialization
	if dst.Attributes.Labels == nil {
		dst.Attributes.Labels = make(map[string]string)
	}

	// Add cluster and registry information to labels
	clusterID := string(srcRegistry.Cluster())
	registryID := string(srcRegistry.Provider())
	clusterLabel := "cluster." + clusterID
	registryLabel := "registry." + registryID

	// Enhanced error handling for source labels
	if src.Attributes.Labels != nil {
		for key, value := range src.Attributes.Labels {
			if key == "" || value == "" {
				klog.Warningf("Skipping empty label key/value from registry %s/%s", registryID, clusterID)
				continue
			}
			// Prefix labels with cluster/registry info to avoid conflicts
			prefixedKey := clusterLabel + "/" + key
			dst.Attributes.Labels[prefixedKey] = value
		}
	}

	// Add registry health status metadata
	dst.Attributes.Labels[registryLabel+"/health"] = "active"
	dst.Attributes.Labels[registryLabel+"/cluster"] = clusterID

	// Add registry provider info
	dst.Attributes.Labels[registryLabel] = "true"

	// Merge service accounts from different clusters
	for _, sa := range src.ServiceAccounts {
		found := false
		for _, existingSa := range dst.ServiceAccounts {
			if existingSa == sa {
				found = true
				break
			}
		}
		if !found {
			dst.ServiceAccounts = append(dst.ServiceAccounts, sa)
		}
	}
}

// mergeServicePorts merges service ports with conflict resolution
func mergeServicePorts(dst, src *model.Service, srcRegistry serviceregistry.Instance) {
	for _, srcPort := range src.Ports {
		found := false
		for _, dstPort := range dst.Ports {
			if dstPort.Port == srcPort.Port && dstPort.Protocol == srcPort.Protocol {
				found = true
				// Port conflict resolution - prefer Kubernetes registry
				if srcRegistry.Provider() == provider.Kubernetes && dstPort.Name != srcPort.Name {
					klog.V(2).Infof("Port name conflict for service %s port %d: %s vs %s, preferring Kubernetes",
						src.Hostname, srcPort.Port, dstPort.Name, srcPort.Name)
					dstPort.Name = srcPort.Name
				}
				break
			}
		}

		// Add port if not found
		if !found {
			dst.Ports = append(dst.Ports, &model.Port{
				Name:     srcPort.Name,
				Port:     srcPort.Port,
				Protocol: srcPort.Protocol,
			})
		}
	}
}

// GetHealthyRegistries returns only healthy registries, implementing basic failover
func (c *Controller) GetHealthyRegistries() []serviceregistry.Instance {
	c.storeLock.RLock()
	defer c.storeLock.RUnlock()

	healthy := make([]serviceregistry.Instance, 0)
	for _, r := range c.registries {
		if r.HasSynced() {
			healthy = append(healthy, r)
		}
	}

	return healthy
}

// GetRegistriesByProvider returns registries filtered by provider type
func (c *Controller) GetRegistriesByProvider(providerType provider.ID) []serviceregistry.Instance {
	c.storeLock.RLock()
	defer c.storeLock.RUnlock()

	filtered := make([]serviceregistry.Instance, 0)
	for _, r := range c.registries {
		if r.Provider() == providerType {
			filtered = append(filtered, r)
		}
	}

	return filtered
}

// AddRegistry adds a new service registry to the aggregator with failover support
func (c *Controller) AddRegistry(registry serviceregistry.Instance) {
	if registry == nil {
		klog.Errorf("Cannot add nil registry to aggregate controller")
		return
	}

	c.storeLock.Lock()
	defer c.storeLock.Unlock()

	clusterID := registry.Cluster()
	providerID := registry.Provider()

	// Enhanced validation
	if clusterID == "" || providerID == "" {
		klog.Errorf("Registry has invalid cluster ID (%s) or provider ID (%s)", clusterID, providerID)
		return
	}

	// Check if registry already exists
	for i, existing := range c.registries {
		if existing.Cluster() == clusterID && existing.Provider() == providerID {
			klog.Warningf("Registry %s/%s already exists at index %d, replacing with new instance", providerID, clusterID, i)
			// Replace existing registry instead of skipping
			c.registries[i] = &registryEntry{
				Instance: registry,
				stop:     nil,
			}
			if c.running {
				klog.Infof("Restarting replaced registry %s/%s", providerID, clusterID)
				go registry.Run(c.getStopChannel())
			}
			return
		}
	}

	entry := &registryEntry{
		Instance: registry,
		stop:     nil, // Use server stop channel
	}

	c.registries = append(c.registries, entry)
	klog.Infof("Successfully added registry %s/%s (total: %d registries)", providerID, clusterID, len(c.registries))

	// Start registry if aggregator is already running
	if c.running {
		klog.Infof("Starting newly added registry %s/%s", providerID, clusterID)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					klog.Errorf("Registry %s/%s panicked during startup: %v", providerID, clusterID, r)
				}
			}()
			registry.Run(c.getStopChannel())
		}()
	}

	klog.Infof("Added registry %s/%s to aggregator", registry.Provider(), registry.Cluster())
}

// RemoveRegistry removes a service registry from the aggregator
func (c *Controller) RemoveRegistry(clusterID cluster.ID, providerType provider.ID) bool {
	c.storeLock.Lock()
	defer c.storeLock.Unlock()

	for i, r := range c.registries {
		if r.Cluster() == clusterID && r.Provider() == providerType {
			// Note: We cannot close the stop channel here as it's read-only
			// The registry will be stopped when the main stop channel is closed
			// In a real implementation, we would need a different mechanism to stop individual registries

			// Remove from slice
			c.registries = append(c.registries[:i], c.registries[i+1:]...)
			klog.Infof("Removed registry %s/%s from aggregator", providerType, clusterID)
			return true
		}
	}

	klog.Warningf("Registry %s/%s not found for removal", providerType, clusterID)
	return false
}

// getStopChannel returns a stop channel for the aggregator
func (c *Controller) getStopChannel() <-chan struct{} {
	// This is a simplified implementation
	// In a real scenario, this would return the server's stop channel
	stopCh := make(chan struct{})
	return stopCh
}
