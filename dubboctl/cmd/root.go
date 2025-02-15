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

package cmd

import (
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/dashboard"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/deploy"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/generate"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/proxy"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/registry"
)

import (
	"github.com/ory/viper"

	"github.com/spf13/cobra"
)

import (
	cmd2 "github.com/apache/dubbo-kubernetes/pkg/core/cmd"
)

type RootCommandConfig struct {
	Name      string
	NewClient deploy.ClientFactory
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(args []string) {
	rootCmd := GetRootCmd(args)
	// when flag error occurs, print usage string.
	// but if an error occurs when executing command, usage string will not be printed.
	rootCmd.SetFlagErrorFunc(func(command *cobra.Command, err error) error {
		command.Println(command.UsageString())

		return err
	})

	cobra.CheckErr(rootCmd.Execute())
}

func GetRootCmd(args []string) *cobra.Command {
	// rootCmd represents the base command when called without any subcommands
	rootCmd := &cobra.Command{
		Use:           "dubboctl",
		Short:         "dubbo control interface",
		Long:          ``,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cfg := RootCommandConfig{
		Name: "dubboctl",
	}

	// DeployMode Variables
	// Evaluated first after static defaults, set all flags to be associated with
	// a version prefixed by "DUBBO_"
	viper.AutomaticEnv()        // read in environment variables for DUBBO_<flag>
	viper.SetEnvPrefix("dubbo") // ensure that all have the prefix
	newClient := cfg.NewClient
	if newClient == nil {
		newClient = deploy.NewClient
	}

	addSubCommands(rootCmd, newClient)
	rootCmd.SetArgs(args)
	return rootCmd
}

func addSubCommands(rootCmd *cobra.Command, newClient deploy.ClientFactory) {
	deploy.AddBuild(rootCmd, newClient)
	deploy.AddCreate(rootCmd, newClient)
	deploy.AddRepository(rootCmd, newClient)
	deploy.AddDeploy(rootCmd, newClient)
	AddManifest(rootCmd)
	generate.AddGenerate(rootCmd)
	addProfile(rootCmd)
	dashboard.AddDashboard(rootCmd)
	registry.AddRegistryCmd(rootCmd)
	proxy.AddProxy(cmd2.DefaultRunCmdOpts, rootCmd)
}
