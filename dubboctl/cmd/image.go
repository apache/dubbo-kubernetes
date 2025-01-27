package cmd

import (
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/spf13/cobra"
)

func ImageCmd(ctx cli.Context, cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	ibc := imageBuildCmd(cmd, clientFactory)
	ipc := imageBuildCmd(cmd, clientFactory)
	iac := imageApplyCmd(cmd, clientFactory)

	ic := &cobra.Command{
		Use:   "image",
		Short: "",
		Long:  "",
	}
	ic.AddCommand(ibc)
	ic.AddCommand(ipc)
	ic.AddCommand(iac)
	return ic
}

func imageBuildCmd(cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	return nil
}

func imagePushCmd(cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	return nil
}

func imageApplyCmd(cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	return nil
}
