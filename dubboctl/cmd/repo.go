package cmd

import (
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/apache/dubbo-kubernetes/operator/cmd/cluster"
	"github.com/spf13/cobra"
)

type repoArgs struct {
}

func addRepoFlags(cmd *cobra.Command, rArgs *repoArgs) {
}

func RepoCmd(_ cli.Context, cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	rootArgs := &cluster.RootArgs{}
	rArgs := &repoArgs{}
	rc := &cobra.Command{
		Use:   "repo",
		Short: "",
		Long:  "",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	cluster.AddFlags(rc, rootArgs)
	addRepoFlags(rc, rArgs)
	return rc
}
