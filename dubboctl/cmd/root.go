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
	"flag"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/validate"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/version"
	"github.com/apache/dubbo-kubernetes/operator/cmd/cluster"
	"github.com/spf13/cobra"
)

func AddFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
}

func GetRootCmd(args []string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "dubboctl",
		Short:        "Dubbo command line utilities",
		SilenceUsage: true,
		Long:         `Dubbo configuration command line utility for debug and use dubbo applications.`,
	}
	AddFlags(rootCmd)
	rootCmd.SetArgs(args)
	flags := rootCmd.PersistentFlags()
	rootOptions := cli.AddRootFlags(flags)
	ctx := cli.NewCLIContext(rootOptions)

	installCmd := cluster.InstallCmd(ctx)
	rootCmd.AddCommand(installCmd)
	hideFlags(installCmd, cli.NamespaceFlag, cli.DubboNamespaceFlag, cli.ChartFlag)

	uninstallCmd := cluster.UninstallCmd(ctx)
	rootCmd.AddCommand(uninstallCmd)

	upgradeCmd := cluster.UpgradeCmd(ctx)
	rootCmd.AddCommand(upgradeCmd)

	manifestCmd := cluster.ManifestCmd(ctx)
	rootCmd.AddCommand(manifestCmd)
	hideFlags(manifestCmd, cli.NamespaceFlag, cli.DubboNamespaceFlag, cli.ChartFlag)

	validateCmd := validate.NewValidateCommand(ctx)
	rootCmd.AddCommand(validateCmd)
	hideFlags(validateCmd, cli.NamespaceFlag, cli.DubboNamespaceFlag, cli.ChartFlag)

	versionCmd := version.NewVersionCommand(ctx)
	rootCmd.AddCommand(versionCmd)
	hideFlags(versionCmd, cli.NamespaceFlag, cli.DubboNamespaceFlag, cli.ChartFlag)

	createCmd := CreateCmd(ctx)
	rootCmd.AddCommand(createCmd)
	hideFlags(createCmd, cli.NamespaceFlag, cli.DubboNamespaceFlag, cli.ChartFlag)

	return rootCmd
}

func hideFlags(origin *cobra.Command, hide ...string) {
	origin.SetHelpFunc(func(command *cobra.Command, args []string) {
		for _, hf := range hide {
			_ = command.Flags().MarkHidden(hf)
		}
		origin.SetHelpFunc(nil)
		origin.HelpFunc()(command, args)
	})
}
