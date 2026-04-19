//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package app

import (
	"context"
	"errors"
	"fmt"
	options "github.com/apache/dubbo-kubernetes/dubbod/agency/options"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/util/network"
	"github.com/apache/dubbo-kubernetes/pkg/dubboagency"
	"github.com/apache/dubbo-kubernetes/pkg/dubboagency/config"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	"github.com/apache/dubbo-kubernetes/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/util/protomarshal"
	"net/netip"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/bootstrap"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/features"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/serviceregistry/provider"
	"github.com/apache/dubbo-kubernetes/pkg/cmd"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/ctrlz"
	"github.com/spf13/cobra"
)

const (
	localHostIPv4 = "127.0.0.1"
	localHostIPv6 = "::1"
)

var (
	serverArgs *bootstrap.DubboArgs
	proxyArgs  options.ProxyArgs
)

func NewRootCommand(sds dubboagency.SDSServiceFactory) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "dubbo-discovery",
		Short:        "Dubbo Control Plane.",
		Long:         "Dubbo Control Plane provides mesh-wide traffic management, security and policy capabilities in the Dubbo Service Mesh.",
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
	proxyCmd := newProxyCommand(sds)
	addFlags(discoveryCmd, proxyCmd)
	rootCmd.AddCommand(discoveryCmd)

	cmd.AddFlags(rootCmd)
	rootCmd.AddCommand(proxyCmd)
	rootCmd.AddCommand(waitCmd)

	return rootCmd
}

func newDiscoveryCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "discovery",
		Short: "dubbo discovery service.",
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

			// Wait for signal - when received, immediately exit
			cmd.WaitSignal(stop)

			// Signal received, exit immediately
			return nil
		},
	}
}

func newProxyCommand(sds dubboagency.SDSServiceFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "proxy",
		Short: "XDS proxy agent",
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
		RunE: func(c *cobra.Command, args []string) error {
			cmd.PrintFlags(c.Flags())

			err := initProxy(args)
			if err != nil {
				return err
			}

			proxyConfig, err := config.ConstructProxyConfig(proxyArgs.MeshGlobalConfigFile, options.ProxyConfigEnv)
			if err != nil {
				return fmt.Errorf("failed to get proxy config: %v", err)
			}

			if out, err := protomarshal.ToYAML(proxyConfig); err != nil {
				log.Infof("Failed to serialize to YAML: %v", err)
			} else {
				log.Infof("Effective config: %s", out)
			}

			secOpts, err := options.NewSecurityOptions(proxyConfig, proxyArgs.StsPort, proxyArgs.TokenManagerPlugin)
			if err != nil {
				return err
			}

			if proxyArgs.TemplateFile != "" && proxyConfig.CustomConfigFile == "" {
				proxyConfig.ProxyBootstrapTemplatePath = proxyArgs.TemplateFile
			}

			agentOptions := options.NewAgentOptions(&proxyArgs, proxyConfig, sds)
			agent := dubboagency.NewAgent(proxyConfig, agentOptions, secOpts)

			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(errors.New("application shutdown"))
			defer agent.Close()

			// On SIGINT or SIGTERM, cancel the context, triggering a graceful shutdown
			go cmd.WaitSignalFunc(cancel)

			wait, err := agent.Run(ctx)
			if err != nil {
				return err
			}
			wait()
			return nil
		},
	}
}

func initProxy(args []string) error {
	proxyArgs.Type = model.Proxyless
	if len(args) > 0 {
		proxyArgs.Type = model.NodeType(args[0])
		if !model.IsApplicationNodeType(proxyArgs.Type) {
			return fmt.Errorf("invalid proxy Type: %s", string(proxyArgs.Type))
		}
	}

	podIP, _ := netip.ParseAddr(options.InstanceIPVar.Get()) // protobuf encoding of IP_ADDRESS type
	if podIP.IsValid() {
		proxyArgs.IPAddresses = []string{podIP.String()}
	}

	proxyAddrs := make([]string, 0)
	if ipAddrs, ok := network.GetPrivateIPs(context.Background()); ok {
		proxyAddrs = append(proxyAddrs, ipAddrs...)
	}

	if len(proxyAddrs) == 0 {
		proxyAddrs = append(proxyAddrs, localHostIPv4, localHostIPv6)
	}

	proxyArgs.IPAddresses = append(proxyArgs.IPAddresses, proxyAddrs...)
	log.Debugf("Proxy IPAddresses: %v", proxyArgs.IPAddresses)

	proxyArgs.DiscoverIPMode()

	proxyArgs.ID = proxyArgs.PodName + "." + proxyArgs.PodNamespace

	proxyArgs.DNSDomain = getDNSDomain(proxyArgs.PodNamespace, proxyArgs.DNSDomain)
	log.WithLabels("ips", proxyArgs.IPAddresses, "type", proxyArgs.Type, "id", proxyArgs.ID, "domain", proxyArgs.DNSDomain).Info("Proxy role")
	return nil
}

func getDNSDomain(podNamespace, domain string) string {
	if len(domain) == 0 {
		domain = podNamespace + ".svc." + constants.DefaultClusterLocalDomain
	}
	return domain
}

func addFlags(c *cobra.Command, proxyCmd *cobra.Command) {
	serverArgs = bootstrap.NewDubboArgs(func(p *bootstrap.DubboArgs) {
		p.CtrlZOptions = ctrlz.DefaultOptions()
		p.InjectionOptions = bootstrap.InjectionOptions{
			InjectionDirectory: "./var/lib/dubbo/inject",
		}
	})
	proxyArgs = options.NewProxyArgs()
	proxyCmd.PersistentFlags().StringVar(&proxyArgs.DNSDomain,
		"domain",
		"",
		"DNS domain suffix. If not provided uses ${POD_NAMESPACE}.svc.cluster.local")
	proxyCmd.PersistentFlags().IntVar(&proxyArgs.StsPort,
		"stsPort",
		0,
		"HTTP Port on which to serve Security Token Service (STS). If zero, STS service will not be provided.")
	proxyCmd.PersistentFlags().StringVar(&proxyArgs.TemplateFile,
		"templateFile",
		"",
		"Go template bootstrap config")
	c.PersistentFlags().StringSliceVar(&serverArgs.RegistryOptions.Registries,
		"registries",
		[]string{string(provider.Kubernetes)},
		fmt.Sprintf("Comma separated list of platform service registries to read from (choose one or more from {%s})",
			provider.Kubernetes))
	c.PersistentFlags().StringVar(&serverArgs.RegistryOptions.ClusterRegistriesNamespace,
		"clusterRegistriesNamespace", serverArgs.RegistryOptions.ClusterRegistriesNamespace,
		"Namespace for ConfigMap which stores clusters configs")
	c.PersistentFlags().StringVar(&serverArgs.RegistryOptions.KubeConfig,
		"kubeconfig",
		"",
		"Use a Kubernetes configuration file instead of in-cluster configuration")
	c.PersistentFlags().StringVar(&serverArgs.MeshGlobalConfigFile,
		"meshGlobalConfig",
		"./etc/dubbo/config/mesh",
		"File name for Dubbo mesh global configuration. If not specified, a default mesh will be used.")
	c.PersistentFlags().Float32Var(&serverArgs.RegistryOptions.KubeOptions.KubernetesAPIQPS,
		"kubernetesApiQPS",
		80.0,
		"Maximum QPS when communicating with the kubernetes API")
	c.PersistentFlags().IntVar(&serverArgs.RegistryOptions.KubeOptions.KubernetesAPIBurst,
		"kubernetesApiBurst",
		160,
		"Maximum burst for throttle when communicating with the kubernetes API")
	c.PersistentFlags().StringVar(&serverArgs.ServerOptions.HTTPAddr,
		"httpAddr",
		":8080",
		"Discovery service HTTP address")
	c.PersistentFlags().StringVar(&serverArgs.ServerOptions.HTTPSAddr,
		"httpsAddr",
		":15017",
		"Injection and validation service HTTPS address")
	c.PersistentFlags().StringVar(&serverArgs.ServerOptions.GRPCAddr,
		"grpcAddr",
		":15010",
		"Discovery service gRPC address")
	c.PersistentFlags().StringVar(&serverArgs.ServerOptions.SecureGRPCAddr,
		"secureGRPCAddr",
		":15012",
		"Discovery service secured gRPC address")
	c.PersistentFlags().StringVar(&serverArgs.RegistryOptions.FileDir,
		"configDir",
		"",
		"Directory to watch for updates to config yaml files. If specified, the files will be used as the source of config, rather than a CRD client.")
	c.PersistentFlags().StringVar(&serverArgs.RegistryOptions.KubeOptions.DomainSuffix,
		"domain", constants.DefaultClusterLocalDomain,
		"DNS domain suffix")
	c.PersistentFlags().StringVarP(&serverArgs.Namespace,
		"namespace",
		"n", serverArgs.Namespace,
		"Select a namespace where the controller resides. If not set, uses ${POD_NAMESPACE} environment variable")
	c.PersistentFlags().StringVar((*string)(&serverArgs.RegistryOptions.KubeOptions.ClusterID),
		"clusterID", features.ClusterName,
		"The ID of the cluster that this Dubbod instance resides")
	c.PersistentFlags().StringToStringVar(&serverArgs.RegistryOptions.KubeOptions.ClusterAliases, "clusterAliases", map[string]string{},
		"Alias names for clusters. Example: alias1=cluster1,alias2=cluster2")

	serverArgs.CtrlZOptions.AttachCobraFlags(c)

	serverArgs.KeepaliveOptions.AttachCobraFlags(c)
}
