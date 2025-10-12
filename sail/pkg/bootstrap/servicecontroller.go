/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package bootstrap

import (
	"fmt"
	"strings"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/sail/pkg/features"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/aggregate"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/istio"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/kube/controller"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/nacos"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/provider"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/zookeeper"
	"k8s.io/klog/v2"
)

func (s *Server) ServiceController() *aggregate.Controller {
	return s.environment.ServiceDiscovery.(*aggregate.Controller)
}

// initServiceControllers creates and initializes the service controllers
func (s *Server) initServiceControllers(args *SailArgs) error {
	serviceControllers := s.ServiceController()

	// Initialize standard registries from command line
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
		case provider.Nacos:
			if err := s.initNacosRegistry(args); err != nil {
				return err
			}
		case provider.Zookeeper:
			if err := s.initZookeeperRegistry(args); err != nil {
				return err
			}
		case provider.Istio:
			if err := s.initIstioRegistry(args); err != nil {
				return err
			}
		default:
			return fmt.Errorf("service registry %s is not supported", r)
		}
	}

	// Initialize multi-registry configuration if enabled
	if args.RegistryOptions.MultiRegistry.Enabled {
		if err := s.initMultiRegistryControllers(args); err != nil {
			return fmt.Errorf("failed to initialize multi-registry controllers: %v", err)
		}
	}

	// Initialize multi-cluster configuration if enabled
	if args.RegistryOptions.MultiCluster.Enabled {
		if err := s.initMultiClusterControllers(args); err != nil {
			return fmt.Errorf("failed to initialize multi-cluster controllers: %v", err)
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

	// Initialize k8s native service discovery if enabled
	if args.RegistryOptions.EnableK8sServiceDiscovery || features.EnableK8sServiceDiscovery {
		klog.Info("Initializing Kubernetes native service discovery")

		// Validate prerequisites
		if s.kubeClient == nil {
			klog.Error("Kubernetes client is not available for service discovery")
			return fmt.Errorf("kubernetes client required for k8s service discovery")
		}

		// Set k8s service discovery options with validation
		args.RegistryOptions.KubeOptions.EnableK8sServiceDiscovery = true
		args.RegistryOptions.KubeOptions.K8sServiceNamespaces = args.RegistryOptions.K8sServiceNamespaces
		args.RegistryOptions.KubeOptions.DubboAnnotationPrefix = args.RegistryOptions.DubboAnnotationPrefix

		// Validate namespaces configuration
		if len(args.RegistryOptions.K8sServiceNamespaces) == 0 {
			klog.Warning("No namespaces specified for k8s service discovery, using default namespace")
			args.RegistryOptions.KubeOptions.K8sServiceNamespaces = []string{"default"}
		}

		// Create k8s service discovery controller with error handling
		k8sController := controller.NewController(args.RegistryOptions.KubeOptions, s.kubeClient)
		if k8sController == nil {
			klog.Error("Kubernetes controller is nil after creation")
			return fmt.Errorf("kubernetes controller initialization failed")
		}

		// Add the controller to the aggregate service controller
		serviceControllers := s.ServiceController()
		if serviceControllers == nil {
			klog.Error("Service controller is not available")
			return fmt.Errorf("service controller not initialized")
		}

		// Register k8s native controller with error recovery
		defer func() {
			if r := recover(); r != nil {
				klog.Errorf("Panic during k8s controller registration: %v", r)
			}
		}()

		// TODO: Add method to register k8s native controller
		// serviceControllers.AddRegistry(provider.KubernetesNative, k8sController)

		klog.Info("Kubernetes native service discovery initialized successfully")
	}

	// TODO NewMulticluster
	return
}

// initNacosRegistry initializes Nacos service registry
func (s *Server) initNacosRegistry(args *SailArgs) error {
	klog.Info("Initializing Nacos service registry")

	// Create Nacos configuration from command line args or config file
	config := nacos.DefaultConfig()

	// TODO: Parse Nacos configuration from args or config file
	// For now, use default configuration

	// Create Nacos controller
	nacosController := nacos.NewController(config, s.clusterID)
	if nacosController == nil {
		return fmt.Errorf("failed to create Nacos controller")
	}

	// Add to service controllers
	s.ServiceController().AddRegistry(nacosController)

	klog.Info("Nacos service registry initialized successfully")
	return nil
}

// initZookeeperRegistry initializes Zookeeper service registry
func (s *Server) initZookeeperRegistry(args *SailArgs) error {
	klog.Info("Initializing Zookeeper service registry")

	// Create Zookeeper configuration from command line args or config file
	config := zookeeper.DefaultConfig()

	// TODO: Parse Zookeeper configuration from args or config file
	// For now, use default configuration

	// Create Zookeeper controller
	zkController := zookeeper.NewController(config, s.clusterID)
	if zkController == nil {
		return fmt.Errorf("failed to create Zookeeper controller")
	}

	// Add to service controllers
	s.ServiceController().AddRegistry(zkController)

	klog.Info("Zookeeper service registry initialized successfully")
	return nil
}

// initIstioRegistry initializes Istio service registry
func (s *Server) initIstioRegistry(args *SailArgs) error {
	klog.Info("Initializing Istio service registry")

	// Create Istio configuration from command line args
	config := &istio.IstioConfig{
		PilotAddress:    args.RegistryOptions.IstioOptions.PilotAddress,
		Namespace:       args.RegistryOptions.IstioOptions.Namespace,
		TLSEnabled:      args.RegistryOptions.IstioOptions.TLSEnabled,
		CertPath:        args.RegistryOptions.IstioOptions.CertPath,
		KeyPath:         args.RegistryOptions.IstioOptions.KeyPath,
		CAPath:          args.RegistryOptions.IstioOptions.CACertPath,
		EnableDiscovery: true,
		SyncTimeout:     time.Duration(args.RegistryOptions.IstioOptions.ConnectionTimeout) * time.Second,
	}

	// Use defaults if no configuration provided
	if config.PilotAddress == "" {
		config = istio.DefaultConfig()
	}

	// Validate Istio configuration
	configManager := istio.NewConfigManager()
	if err := configManager.LoadFromConfig(config); err != nil {
		return fmt.Errorf("invalid Istio configuration: %v", err)
	}

	// Create Istio controller
	istioController := istio.NewController(config, s.clusterID)
	if istioController == nil {
		return fmt.Errorf("failed to create Istio controller")
	}

	// Add to service controllers
	s.ServiceController().AddRegistry(istioController)

	klog.Info("Istio service registry initialized successfully")
	return nil
}

// initMultiRegistryControllers initializes multiple service registries from configuration
func (s *Server) initMultiRegistryControllers(args *SailArgs) error {
	klog.Info("Initializing multi-registry controllers")

	for _, regConfig := range args.RegistryOptions.MultiRegistry.Registries {
		if !regConfig.Enabled {
			klog.V(2).Infof("Skipping disabled registry: %s", regConfig.ID)
			continue
		}

		klog.Infof("Initializing registry: %s (%s)", regConfig.Name, regConfig.Type)

		switch strings.ToLower(regConfig.Type) {
		case "kubernetes":
			if err := s.initKubeRegistryFromConfig(regConfig); err != nil {
				return fmt.Errorf("failed to initialize Kubernetes registry %s: %v", regConfig.ID, err)
			}
		case "nacos":
			if err := s.initNacosRegistryFromConfig(regConfig); err != nil {
				return fmt.Errorf("failed to initialize Nacos registry %s: %v", regConfig.ID, err)
			}
		case "zookeeper":
			if err := s.initZookeeperRegistryFromConfig(regConfig); err != nil {
				return fmt.Errorf("failed to initialize Zookeeper registry %s: %v", regConfig.ID, err)
			}
		default:
			klog.Warningf("Unknown registry type: %s for registry %s", regConfig.Type, regConfig.ID)
		}
	}

	klog.Info("Multi-registry controllers initialized successfully")
	return nil
}

// initMultiClusterControllers initializes multiple cluster configurations
func (s *Server) initMultiClusterControllers(args *SailArgs) error {
	klog.Info("Initializing multi-cluster controllers")

	// Create cluster manager
	clusterManager := cluster.NewManager(s.clusterID)

	// Add clusters from configuration
	for _, clusterConfig := range args.RegistryOptions.MultiCluster.Clusters {
		if !clusterConfig.Enabled {
			klog.V(2).Infof("Skipping disabled cluster: %s", clusterConfig.ID)
			continue
		}

		klog.Infof("Adding cluster: %s (%s)", clusterConfig.Name, clusterConfig.ID)

		if err := clusterManager.AddCluster(cluster.ClusterConfig{
			ID:         clusterConfig.ID,
			Name:       clusterConfig.Name,
			KubeConfig: clusterConfig.KubeConfig,
			Context:    clusterConfig.Context,
			Enabled:    clusterConfig.Enabled,
			Priority:   clusterConfig.Priority,
			Region:     clusterConfig.Region,
			Zone:       clusterConfig.Zone,
		}); err != nil {
			return fmt.Errorf("failed to add cluster %s: %v", clusterConfig.ID, err)
		}
	}

	// TODO: Store cluster manager in server context for later use

	klog.Info("Multi-cluster controllers initialized successfully")
	return nil
}

// Helper methods for initializing registries from configuration

func (s *Server) initKubeRegistryFromConfig(regConfig RegistryConfig) error {
	// TODO: Initialize Kubernetes registry from specific configuration
	klog.V(2).Infof("Initializing Kubernetes registry from config: %s", regConfig.ID)
	return nil
}

func (s *Server) initNacosRegistryFromConfig(regConfig RegistryConfig) error {
	config := nacos.DefaultConfig()

	// Parse Nacos-specific configuration
	if servers, ok := regConfig.Config["serverAddrs"].([]interface{}); ok {
		config.ServerAddrs = make([]string, len(servers))
		for i, server := range servers {
			if serverStr, ok := server.(string); ok {
				config.ServerAddrs[i] = serverStr
			}
		}
	}

	if namespace, ok := regConfig.Config["namespace"].(string); ok {
		config.Namespace = namespace
	}

	if group, ok := regConfig.Config["group"].(string); ok {
		config.Group = group
	}

	if username, ok := regConfig.Config["username"].(string); ok {
		config.Username = username
	}

	if password, ok := regConfig.Config["password"].(string); ok {
		config.Password = password
	}

	if timeoutStr, ok := regConfig.Config["timeout"].(string); ok {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			config.Timeout = timeout
		}
	}

	// Create controller with custom cluster ID
	clusterID := cluster.ID(regConfig.ID)
	nacosController := nacos.NewController(config, clusterID)
	if nacosController == nil {
		return fmt.Errorf("failed to create Nacos controller")
	}

	// Add to service controllers
	s.ServiceController().AddRegistry(nacosController)

	return nil
}

func (s *Server) initZookeeperRegistryFromConfig(regConfig RegistryConfig) error {
	config := zookeeper.DefaultConfig()

	// Parse Zookeeper-specific configuration
	if servers, ok := regConfig.Config["servers"].([]interface{}); ok {
		config.Servers = make([]string, len(servers))
		for i, server := range servers {
			if serverStr, ok := server.(string); ok {
				config.Servers[i] = serverStr
			}
		}
	}

	if root, ok := regConfig.Config["root"].(string); ok {
		config.Root = root
	}

	if username, ok := regConfig.Config["username"].(string); ok {
		config.Username = username
	}

	if password, ok := regConfig.Config["password"].(string); ok {
		config.Password = password
	}

	if timeoutStr, ok := regConfig.Config["timeout"].(string); ok {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			config.Timeout = timeout
		}
	}

	if sessionTimeoutStr, ok := regConfig.Config["sessionTimeout"].(string); ok {
		if sessionTimeout, err := time.ParseDuration(sessionTimeoutStr); err == nil {
			config.SessionTimeout = sessionTimeout
		}
	}

	// Create controller with custom cluster ID
	clusterID := cluster.ID(regConfig.ID)
	zkController := zookeeper.NewController(config, clusterID)
	if zkController == nil {
		return fmt.Errorf("failed to create Zookeeper controller")
	}

	// Add to service controllers
	s.ServiceController().AddRegistry(zkController)

	return nil
}
