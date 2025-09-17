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
	"github.com/apache/dubbo-kubernetes/pkg/model"
	"github.com/apache/dubbo-kubernetes/ship/cmd/ship-agent/options"
	"github.com/spf13/cobra"
)

var (
	proxyArgs options.ProxyArgs
)

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "ship-agent",
		Short:        "Dubbo Ship agent.",
		Long:         "Dubbo Ship agent runs in the sidecar or gateway container and bootstraps Envoy.",
		SilenceUsage: true,
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			// Allow unknown flags for backward-compatibility.
			UnknownFlags: true,
		},
	}
	cmd.AddFlags(rootCmd)
	proxyCmd := newProxyCommand()
	addFlags(proxyCmd)
	rootCmd.AddCommand(proxyCmd)
	rootCmd.AddCommand(waitCmd)

	return rootCmd
}

func newProxyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "proxy",
		Short: "XDS proxy agent",
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			// Allow unknown flags for backward-compatibility.
			UnknownFlags: true,
		},
		RunE: func(c *cobra.Command, args []string) error {
			cmd.PrintFlags(c.Flags())
			err := initProxy(args)
			if err != nil {
				return err
			}
			return nil
		},
	}
}

func initProxy(args []string) error {
	proxyArgs.Type = model.SidecarProxy
	if len(args) > 0 {
		proxyArgs.Type = model.NodeType(args[0])
		if !model.IsApplicationNodeType(proxyArgs.Type) {
			return fmt.Errorf("invalid proxy Type: %s", string(proxyArgs.Type))
		}
	}
	return nil
}

func addFlags(proxyCmd *cobra.Command) {
	proxyArgs = options.NewProxyArgs()
	proxyCmd.PersistentFlags().StringVar(&proxyArgs.DNSDomain, "domain", "",
		"DNS domain suffix. If not provided uses ${POD_NAMESPACE}.svc.cluster.local")
	proxyCmd.PersistentFlags().StringVar(&proxyArgs.MeshConfigFile, "meshConfig", "./etc/dubbo/config/mesh",
		"File name for Dubbo mesh configuration. If not specified, a default mesh will be used. This may be overridden by "+
			"PROXY_CONFIG environment variable or proxy.dubbo.io/config annotation.")
	proxyCmd.PersistentFlags().StringVar(&proxyArgs.ServiceCluster, "serviceCluster", constants.ServiceClusterName, "Service cluster")
}
