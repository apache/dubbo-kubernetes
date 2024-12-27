package cluster

import (
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util/clog"

	//"github.com/apache/dubbo-kubernetes/operator/pkg/util/clog"
	"github.com/spf13/cobra"
)

type upgradeArgs struct {
	*installArgs
}

func UpgradeCmd(ctx cli.Context) *cobra.Command {
	rootArgs := &RootArgs{}
	upArgs := &upgradeArgs{
		installArgs: &installArgs{},
	}
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade the Dubbo Control Plane",
		Long:  "The upgrade command can be used instead of the install command",
		Example: `  # Apply a default dubboctl installation.
  dubboctl upgrade
  
  # Apply a config file.
  dubboctl upgrade -f dop.yaml
  
  # Apply a default profile.
  dubboctl upgrade --profile=demo

		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cl := clog.NewConsoleLogger(cmd.OutOrStdout(), cmd.ErrOrStderr(), installerScope)
			p := NewPrinterForWriter(cmd.OutOrStderr())
			client, err := ctx.CLIClient()
			if err != nil {
				return err
			}
			return Install(client, rootArgs, upArgs.installArgs, cl, cmd.OutOrStdout(), p)
		},
	}
	addFlags(cmd, rootArgs)
	addInstallFlags(cmd, upArgs.installArgs)
	return cmd
}
