// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package registry

import (
	"github.com/spf13/cobra"
)

import (
	"github.com/apache/dubbo-kubernetes/dubboctl/operator/registry/zk"
)

func AddRegistryCmd(rootCmd *cobra.Command) {
	addZkRegistryCmd(rootCmd)
}

func addZkRegistryCmd(rootCmd *cobra.Command) {
	zkRegistryCmd := &cobra.Command{
		Use:   "zk",
		Short: "Commands related to zookeeper registry",
		Long:  "Commands help user to operate zookeeper registry",
	}
	addZkLsCmd(zkRegistryCmd)
	rootCmd.AddCommand(zkRegistryCmd)
}

func addZkLsCmd(zkRegistryCmd *cobra.Command) {
	zkAddr := "127.0.0.1:2181"

	lsCmd := &cobra.Command{
		Use:   "ls",
		Short: "List services or instances",
		Long: "List all services or list instances of a service\n\n" +
			"Usage:\n\n" +
			"- List all services\n" +
			"  dubboctl zk ls\n\n" +
			"- List instances of a service\n" +
			"  dubboctl zk ls [APP_NAME] [SERVICE_NAME]\n\n" +
			"Example:\n\n" +
			"- List all services\n" +
			"  dubboctl zk ls\n\n" +
			"- List instances of a service\n" +
			"  dubboctl zk ls dubbo com.apache.dubbo.sample.basic.IGreeter\n",
		Args: cobra.MaximumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			// if no args, list all services
			if len(args) == 0 {
				registry, err := zk.NewZkRegistry(zkAddr)
				if err != nil {
					cmd.Println(err)
					return
				}
				services, err := registry.ListServices(nil)
				if err != nil {
					cmd.Println(err)
					return
				}
				for _, service := range services {
					cmd.Println(service)
				}
				return
			}
			if len(args) == 2 {
				registry, err := zk.NewZkRegistry(zkAddr)
				if err != nil {
					cmd.Println(err)
					return
				}
				instances, err := registry.ListInstances(nil, args[0], args[1])
				if err != nil {
					cmd.Println(err)
					return
				}
				for _, instance := range instances {
					cmd.Println(instance)
				}
				return
			}
			cmd.Println("Invalid args, you can use `dubboctl zk ls` to list all services or use `dubboctl zk ls [APP_NAME] [SERVICE_NAME]` to list instances of a service")
		},
	}

	lsCmd.Flags().StringVarP(&zkAddr, "addr", "a", "127.0.0.1:2181", "zookeeper address, if has multiple address, use comma to separate")
	zkRegistryCmd.AddCommand(lsCmd)
}
