package bootstrap

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/aggregate"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/provider"
	"k8s.io/klog/v2"
)

func (s *Server) ServiceController() *aggregate.Controller {
	return s.environment.ServiceDiscovery.(*aggregate.Controller)
}

// initServiceControllers creates and initializes the service controllers
func (s *Server) initServiceControllers(args *SailArgs) error {
	serviceControllers := s.ServiceController()

	registered := sets.New[provider.ID]()
	for _, r := range args.RegistryOptions.Registries {
		serviceRegistry := provider.ID(r)
		if registered.Contains(serviceRegistry) {
			klog.Infof("%s registry specified multiple times.", r)
			continue
		}
		registered.Insert(serviceRegistry)
		klog.Infof("Adding %s registry adapter", serviceRegistry)
		switch serviceRegistry {
		case provider.Kubernetes:
			if err := s.initKubeRegistry(args); err != nil {
				return err
			}
		default:
			return fmt.Errorf("service registry %s is not supported", r)
		}
	}

	// Defer running of the service controllers.
	s.addStartFunc("service controllers", func(stop <-chan struct{}) error {
		go serviceControllers.Run(stop)
		return nil
	})

	return nil
}

// initKubeRegistry creates all the k8s service controllers under this pilot
func (s *Server) initKubeRegistry(args *SailArgs) (err error) {
	args.RegistryOptions.KubeOptions.ClusterID = s.clusterID
	// TODO reversion
	args.RegistryOptions.KubeOptions.KrtDebugger = args.KrtDebugger
	// TODO metrics
	args.RegistryOptions.KubeOptions.XDSUpdater = s.XDSServer
	args.RegistryOptions.KubeOptions.MeshNetworksWatcher = s.environment.NetworksWatcher
	args.RegistryOptions.KubeOptions.MeshWatcher = s.environment.Watcher
	args.RegistryOptions.KubeOptions.SystemNamespace = args.Namespace
	args.RegistryOptions.KubeOptions.MeshServiceController = s.ServiceController()
	// TODO NewMulticluster
	return
}
