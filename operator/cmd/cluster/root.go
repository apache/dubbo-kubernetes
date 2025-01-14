package cluster

import "github.com/spf13/cobra"

type RootFlags struct{}

type RootArgs struct {
	DryRun bool
	RootFlags
}

func AddFlags(cmd *cobra.Command, rootArgs *RootArgs) {
	cmd.Flags().BoolVar(&rootArgs.DryRun, "dry-run", false, `Outputs only the console/log without making any changes`)
}
