package cluster

import (
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/spf13/cobra"
)

func ProfileCmd(ctx cli.Context) *cobra.Command {
	rootArgs := &RootArgs{}
	pc := &cobra.Command{
		Use:   "profile",
		Short: "Commands related to Dubbo configuration profiles",
		Long:  "The profile command lists, dumps or diffs Dubbo configuration profiles.",
		Example: "dubboctl profile list\n" +
			"dubboctl install --set profile=demo  # Use a profile from the list",
	}
	AddFlags(pc, rootArgs)
	return pc
}
