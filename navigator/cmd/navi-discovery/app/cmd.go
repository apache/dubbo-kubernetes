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
	"github.com/apache/dubbo-kubernetes/navigator/pkg/bootstrap"
	"github.com/apache/dubbo-kubernetes/pkg/cmd"
	"github.com/spf13/cobra"
)

var (
	serverArgs *bootstrap.NaviArgs
)

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "navi-discovery",
		Short:        "Dubbo Navigator.",
		Long:         "Dubbo Navigator provides mesh-wide traffic management, security and policy capabilities in the Dubbo Service Mesh.",
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
			// Create the stop channel for all the servers.
			stop := make(chan struct{})

			// Create the server for the discovery service.
			discoveryServer, err := bootstrap.NewServer(serverArgs)
			if err != nil {
				return fmt.Errorf("failed to create discovery service: %v", err)
			}

			// Start the server
			if err := discoveryServer.Start(stop); err != nil {
				return fmt.Errorf("failed to start discovery service: %v", err)
			}
			return nil
		},
	}
}

func addFlags(c *cobra.Command) {
	c.PersistentFlags().StringVar(&serverArgs.RegistryOptions.KubeConfig, "kubeconfig", "",
		"Use a Kubernetes configuration file instead of in-cluster configuration")
}
