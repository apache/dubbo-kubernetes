package cluster

import "github.com/spf13/cobra"

type RootFlags struct {
	kubeconfig     *string
	namespace      *string
	dubboNamespace *string
}

type RootArgs struct {
	DryRun bool
	RootFlags
}

func AddRootFlags(cmd *cobra.Command) *RootFlags {
	rootFlags := &RootFlags{
		kubeconfig:     nil,
		namespace:      nil,
		dubboNamespace: nil,
	}
	return rootFlags
}

func AddFlags(cmd *cobra.Command, rootArgs *RootArgs) {
	cmd.Flags().BoolVar(&rootArgs.DryRun, "dry-run", false, "")
}
