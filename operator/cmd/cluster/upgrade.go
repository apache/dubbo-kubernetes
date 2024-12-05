package cluster

import (
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/spf13/cobra"
)

type upgradeArgs struct {
	*installArgs
}

func UpgradeCmd(ctx cli.Context) *cobra.Command {
	return nil
}
