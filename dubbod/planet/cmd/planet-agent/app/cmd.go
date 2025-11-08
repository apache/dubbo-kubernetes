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
	"context"
	"errors"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	"net/netip"

	"github.com/apache/dubbo-kubernetes/dubbod/planet/cmd/planet-agent/options"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/util/network"
	"github.com/apache/dubbo-kubernetes/pkg/cmd"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	dubboagent "github.com/apache/dubbo-kubernetes/pkg/dubbo-agent"
	"github.com/apache/dubbo-kubernetes/pkg/dubbo-agent/config"
	"github.com/apache/dubbo-kubernetes/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/util/protomarshal"
	"github.com/spf13/cobra"
)

const (
	localHostIPv4 = "127.0.0.1"
	localHostIPv6 = "::1"
)

var (
	proxyArgs options.ProxyArgs
)

func NewRootCommand(sds dubboagent.SDSServiceFactory) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "planet-agent",
		Short:        "Dubbo Planet agent.",
		Long:         "Dubbo Planet agent bootstraps via gRPC xDS.",
		SilenceUsage: true,
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
	}
	cmd.AddFlags(rootCmd)
	proxyCmd := newProxyCommand(sds)
	addFlags(proxyCmd)
	rootCmd.AddCommand(proxyCmd)
	rootCmd.AddCommand(waitCmd)
	return rootCmd
}

func newProxyCommand(sds dubboagent.SDSServiceFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "proxy",
		Short: "xDS proxy agent",
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
		RunE: func(c *cobra.Command, args []string) error {
			cmd.PrintFlags(c.Flags())

			err := initProxy(args)
			if err != nil {
				return err
			}

			proxyConfig, err := config.ConstructProxyConfig(proxyArgs.MeshConfigFile, options.ProxyConfigEnv)
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
			agent := dubboagent.NewAgent(proxyConfig, agentOptions, secOpts)
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

func addFlags(proxyCmd *cobra.Command) {
	proxyArgs = options.NewProxyArgs()
	proxyCmd.PersistentFlags().StringVar(&proxyArgs.DNSDomain, "domain", "",
		"DNS domain suffix. If not provided uses ${POD_NAMESPACE}.svc.cluster.local")
	proxyCmd.PersistentFlags().IntVar(&proxyArgs.StsPort, "stsPort", 0,
		"HTTP Port on which to serve Security Token Service (STS). If zero, STS service will not be provided.")
	proxyCmd.PersistentFlags().StringVar(&proxyArgs.TemplateFile, "templateFile", "",
		"Go template bootstrap config")
	proxyCmd.PersistentFlags().StringVar(&proxyArgs.MeshConfigFile, "meshConfig", "./etc/dubbo/config/mesh",
		"File name for Dubbo mesh configuration. If not specified, a default mesh will be used. This may be overridden by "+
			"PROXY_CONFIG environment variable or proxy.dubbo.apache.org/config annotation.")
}
