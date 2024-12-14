package cluster

import (
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	//"github.com/apache/dubbo-kubernetes/operator/pkg/util/clog"
	"github.com/spf13/cobra"
)

type upgradeArgs struct {
	*installArgs
}

func UpgradeCmd(ctx cli.Context) *cobra.Command {
	//rootArgs := &RootArgs{}
	//upArgs := &upgradeArgs{
	//	installArgs: &installArgs{},
	//}
	cmd := &cobra.Command{
		Use:     "upgrade",
		Short:   "Upgrade the Dubbo Control Plane",
		Long:    "",
		Example: "",
		//RunE: func(cmd *cobra.Command, args []string) error {
		//	cl := clog.NewConsoleLogger(cmd.OutOrStdout(), cmd.ErrOrStderr(), installerScope)
		//	p := NewPrinterForWriter(cmd.OutOrStderr())
		//	client, err := ctx.CLIClient()
		//	if err != nil {
		//		return err
		//	}
		//	return nil
		//},
	}
	return cmd
}
