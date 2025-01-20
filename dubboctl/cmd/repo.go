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
		Short: "Manage existing Dubbo SDK module libraries",
		Long:  "The repo command Manage existing Dubbo SDK module libraries",
		Example: `  # Add a new template library.
  dubboctl repo add [name] [URL]
	
  # View the list of template library.
  dubboctl repo list
	
  # Remove an existing template library.
  dubboctl repo remove [name]
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	cluster.AddFlags(rc, rootArgs)
	addRepoFlags(rc, rArgs)
	rc.AddCommand(newRepoAdd(clientFactory))
	rc.AddCommand(newRepoList(clientFactory))
	rc.AddCommand(newRepoRemove(clientFactory))
	return rc
}

func runRepo() {

}

func newRepoAdd(clientFactory ClientFactory) *cobra.Command {
	return nil

}

func newRepoList(clientFactory ClientFactory) *cobra.Command {
	return nil

}

func newRepoRemove(clientFactory ClientFactory) *cobra.Command {
	return nil
}
