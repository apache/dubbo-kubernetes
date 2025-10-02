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
	"github.com/apache/dubbo-kubernetes/pkg/cmd"
	dubboagent "github.com/apache/dubbo-kubernetes/pkg/dubbo-agent"
	"github.com/apache/dubbo-kubernetes/pkg/dubbo-agent/config"
	"github.com/apache/dubbo-kubernetes/pkg/util/protomarshal"
	"github.com/apache/dubbo-kubernetes/sail/cmd/sail-agent/options"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

var (
	proxyArgs options.ProxyArgs
)

func NewRootCommand(sds dubboagent.SDSServiceFactory) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "sail-agent",
		Short:        "Dubbo Sail agent.",
		Long:         "Dubbo Sail agent bootstraps via gRPC xDS.",
		SilenceUsage: true,
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			// Allow unknown flags for backward-compatibility.
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
		Short: "XDS proxy agent",
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			// Allow unknown flags for backward-compatibility.
			UnknownFlags: true,
		},
		RunE: func(c *cobra.Command, args []string) error {
			cmd.PrintFlags(c.Flags())

			proxyConfig, err := config.ConstructProxyConfig(proxyArgs.MeshConfigFile, options.ProxyConfigEnv)
			if err != nil {
				return fmt.Errorf("failed to get proxy config: %v", err)
			}
			if out, err := protomarshal.ToYAML(proxyConfig); err != nil {
				klog.Infof("Failed to serialize to YAML: %v", err)
			} else {
				klog.Infof("Effective config: \n%s", out)
			}

			secOpts, err := options.NewSecurityOptions(proxyConfig, proxyArgs.StsPort, proxyArgs.TokenManagerPlugin)
			if err != nil {
				return err
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

func addFlags(proxyCmd *cobra.Command) {
	proxyArgs = options.NewProxyArgs()
	proxyCmd.PersistentFlags().StringVar(&proxyArgs.DNSDomain, "domain", "",
		"DNS domain suffix. If not provided uses ${POD_NAMESPACE}.svc.cluster.local")
	proxyCmd.PersistentFlags().StringVar(&proxyArgs.MeshConfigFile, "meshConfig", "./etc/dubbo/config/mesh",
		"File name for Dubbo mesh configuration. If not specified, a default mesh will be used. This may be overridden by "+
			"PROXY_CONFIG environment variable or proxy.dubbo.io/config annotation.")
}
