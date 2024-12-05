package cluster

import (
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/spf13/cobra"
)

type uninstallArgs struct {
}

func addUninstallFlags() {

}

func UninstallCmd(ctx cli.Context) *cobra.Command {
	rootArgs := &RootArgs{}
	uiArgs := &uninstallArgs{}
	uicmd := &cobra.Command{
		Use:     "uninstall",
		Short:   "Uninstall Dubbo-related resources",
		Long:    "",
		Example: ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return uninstall(cmd, ctx, rootArgs, uiArgs)
		},
	}
	AddFlags(uicmd, rootArgs)
	addUninstallFlags()
	return uicmd
}

func uninstall(
	cmd *cobra.Command,
	ctx cli.Context,
	rootArgs *RootArgs,
	uiArgs *uninstallArgs) error {
	return nil
}
