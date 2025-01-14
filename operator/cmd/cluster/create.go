package cluster

import (
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/spf13/cobra"
)

type createArgs struct {
	skipConfirmation bool
}

func addCreateFlags(cmd *cobra.Command, args *uninstallArgs) {
	cmd.PersistentFlags().BoolVarP(&args.skipConfirmation, "skip-confirmation", "y", false, `The skipConfirmation determines whether the user is prompted for confirmation.`)
}

func CreateCmd(ctx cli.Context) *cobra.Command {
	cc := &cobra.Command{
		Use:     "create",
		Short:   "",
		Long:    "",
		Example: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	return cc
}
