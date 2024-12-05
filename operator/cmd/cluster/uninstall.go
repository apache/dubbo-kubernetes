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
	return nil
}
