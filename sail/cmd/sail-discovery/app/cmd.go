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

package app

import (
	"fmt"

	"github.com/apache/dubbo-kubernetes/pkg/cmd"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/ctrlz"
	"github.com/apache/dubbo-kubernetes/sail/pkg/bootstrap"
	"github.com/apache/dubbo-kubernetes/sail/pkg/features"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/provider"
	"github.com/spf13/cobra"
)

var (
	serverArgs *bootstrap.SailArgs

	// Multi-cluster configuration variables
	multiClusterConfigFile string

	// Multi-registry configuration variables
	multiRegistryConfigFile string

	// Nacos configuration variables
	nacosServerAddrs string
	nacosNamespace   string
	nacosGroup       string
	nacosUsername    string
	nacosPassword    string

	// Zookeeper configuration variables
	zookeeperServers  string
	zookeeperRoot     string
	zookeeperUsername string
	zookeeperPassword string
)

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "sail-discovery",
		Short:        "Dubbo Sail.",
		Long:         "Dubbo Sail provides mesh-wide traffic management, security and policy capabilities in the Dubbo Service Mesh.",
		SilenceUsage: true,
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			// Allow unknown flags for backward-compatibility.
			UnknownFlags: true,
		},
		PreRunE: func(c *cobra.Command, args []string) error {
			cmd.AddFlags(c)
			return nil
		},
	}
	discoveryCmd := newDiscoveryCommand()
	addFlags(discoveryCmd)
	rootCmd.AddCommand(discoveryCmd)
	return rootCmd
}

func newDiscoveryCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "discovery",
		Short: "Start Dubbo proxy discovery service.",
		Args:  cobra.ExactArgs(0),
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			// Allow unknown flags for backward-compatibility.
			UnknownFlags: true,
		},
		PreRunE: func(c *cobra.Command, args []string) error {
			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			cmd.PrintFlags(c.Flags())
			// Create the stop channel for all the servers.
			stop := make(chan struct{})

			discoveryServer, err := bootstrap.NewServer(serverArgs)
			if err != nil {
				return fmt.Errorf("failed to create discovery service: %v", err)
			}

			if err := discoveryServer.Start(stop); err != nil {
				return fmt.Errorf("failed to start discovery service: %v", err)
			}
			return nil
		},
	}
}

func addFlags(c *cobra.Command) {
	serverArgs = bootstrap.NewSailArgs(func(p *bootstrap.SailArgs) {
		p.CtrlZOptions = ctrlz.DefaultOptions()
	})
	c.PersistentFlags().StringSliceVar(&serverArgs.RegistryOptions.Registries, "registries",
		[]string{string(provider.Kubernetes)},
		fmt.Sprintf("Comma separated list of platform service registries to read from (choose one or more from {%s})",
			provider.Kubernetes))
	c.PersistentFlags().StringVar(&serverArgs.RegistryOptions.ClusterRegistriesNamespace, "clusterRegistriesNamespace",
		serverArgs.RegistryOptions.ClusterRegistriesNamespace, "Namespace for ConfigMap which stores clusters configs")
	c.PersistentFlags().StringVar(&serverArgs.RegistryOptions.KubeConfig, "kubeconfig", "",
		"Use a Kubernetes configuration file instead of in-cluster configuration")
	c.PersistentFlags().StringVar(&serverArgs.MeshConfigFile, "meshConfig", "./etc/dubbo/config/mesh",
		"File name for Dubbo mesh configuration. If not specified, a default mesh will be used.")
	c.PersistentFlags().StringVar(&serverArgs.NetworksConfigFile, "networksConfig", "./etc/dubbo/config/meshNetworks",
		"File name for Dubbo mesh networks configuration. If not specified, a default mesh networks will be used.")
	c.PersistentFlags().Float32Var(&serverArgs.RegistryOptions.KubeOptions.KubernetesAPIQPS, "kubernetesApiQPS", 80.0,
		"Maximum QPS when communicating with the kubernetes API")
	c.PersistentFlags().IntVar(&serverArgs.RegistryOptions.KubeOptions.KubernetesAPIBurst, "kubernetesApiBurst", 160,
		"Maximum burst for throttle when communicating with the kubernetes API")
	c.PersistentFlags().StringVar(&serverArgs.ServerOptions.HTTPAddr, "httpAddr", ":8080",
		"Discovery service HTTP address")
	c.PersistentFlags().StringVar(&serverArgs.ServerOptions.HTTPSAddr, "httpsAddr", ":15017",
		"Injection and validation service HTTPS address")
	c.PersistentFlags().StringVar(&serverArgs.ServerOptions.GRPCAddr, "grpcAddr", ":15010",
		"Discovery service gRPC address")
	c.PersistentFlags().StringVar(&serverArgs.ServerOptions.SecureGRPCAddr, "secureGRPCAddr", ":15012",
		"Discovery service secured gRPC address")
	c.PersistentFlags().StringVar(&serverArgs.RegistryOptions.FileDir, "configDir", "",
		"Directory to watch for updates to config yaml files. If specified, the files will be used as the source of config, rather than a CRD client.")
	c.PersistentFlags().StringVar(&serverArgs.RegistryOptions.KubeOptions.DomainSuffix, "domain", constants.DefaultClusterLocalDomain,
		"DNS domain suffix")
	c.PersistentFlags().StringVarP(&serverArgs.Namespace, "namespace", "n", bootstrap.PodNamespace,
		"Select a namespace where the controller resides. If not set, uses ${POD_NAMESPACE} environment variable")
	c.PersistentFlags().StringVar((*string)(&serverArgs.RegistryOptions.KubeOptions.ClusterID), "clusterID", features.ClusterName,
		"The ID of the cluster that this Dubbod instance resides")
	c.PersistentFlags().StringToStringVar(&serverArgs.RegistryOptions.KubeOptions.ClusterAliases, "clusterAliases", map[string]string{},
		"Alias names for clusters. Example: alias1=cluster1,alias2=cluster2")

	// Multi-cluster configuration flags
	c.PersistentFlags().BoolVar(&serverArgs.RegistryOptions.MultiCluster.Enabled, "multiClusterEnabled", false,
		"Enable multi-cluster support")
	c.PersistentFlags().StringVar(&multiClusterConfigFile, "multiClusterConfig", "",
		"Path to multi-cluster configuration file")

	// Multi-registry configuration flags
	c.PersistentFlags().BoolVar(&serverArgs.RegistryOptions.MultiRegistry.Enabled, "multiRegistryEnabled", false,
		"Enable multi-registry support")
	c.PersistentFlags().StringVar(&multiRegistryConfigFile, "multiRegistryConfig", "",
		"Path to multi-registry configuration file")

	// Individual registry configuration flags
	c.PersistentFlags().StringVar(&nacosServerAddrs, "nacosServers", "",
		"Comma separated list of Nacos server addresses (host:port)")
	c.PersistentFlags().StringVar(&nacosNamespace, "nacosNamespace", "public",
		"Nacos namespace to use")
	c.PersistentFlags().StringVar(&nacosGroup, "nacosGroup", "DEFAULT_GROUP",
		"Nacos group to use")
	c.PersistentFlags().StringVar(&nacosUsername, "nacosUsername", "",
		"Username for Nacos authentication")
	c.PersistentFlags().StringVar(&nacosPassword, "nacosPassword", "",
		"Password for Nacos authentication")

	c.PersistentFlags().StringVar(&zookeeperServers, "zookeeperServers", "",
		"Comma separated list of Zookeeper server addresses (host:port)")
	c.PersistentFlags().StringVar(&zookeeperRoot, "zookeeperRoot", "/dubbo",
		"Root path for Dubbo services in Zookeeper")
	c.PersistentFlags().StringVar(&zookeeperUsername, "zookeeperUsername", "",
		"Username for Zookeeper authentication")
	c.PersistentFlags().StringVar(&zookeeperPassword, "zookeeperPassword", "",
		"Password for Zookeeper authentication")

	// Kubernetes native service discovery flags
	c.PersistentFlags().BoolVar(&serverArgs.RegistryOptions.EnableK8sServiceDiscovery, "enable-k8s-service-discovery", false,
		"Enable Kubernetes native service discovery for Dubbo services")
	c.PersistentFlags().StringSliceVar(&serverArgs.RegistryOptions.K8sServiceNamespaces, "k8s-service-namespaces", []string{"default"},
		"Namespaces to watch for Kubernetes services with Dubbo annotations")
	c.PersistentFlags().StringVar(&serverArgs.RegistryOptions.DubboAnnotationPrefix, "dubbo-annotation-prefix", "dubbo.apache.org",
		"Prefix for Dubbo service annotations on Kubernetes services")
}
