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
	ad := addCmd(cmd, clientFactory)
	li := listCmd(cmd, clientFactory)
	re := removeCmd(cmd, clientFactory)
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
			return runRepo(cmd, args, clientFactory)
		},
	}
	cluster.AddFlags(rc, rootArgs)
	addRepoFlags(rc, rArgs)
	rc.AddCommand(ad)
	rc.AddCommand(li)
	rc.AddCommand(re)
	return rc
}

func runRepo(cmd *cobra.Command, args []string, clientFactory ClientFactory) error {
	return nil
}

func addCmd(cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	ac := &cobra.Command{
		Use: "add",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdd(cmd, args, clientFactory)
		},
	}
	return ac
}

func runAdd(_ *cobra.Command, args []string, clientFactory ClientFactory) error {
	return nil
}

func listCmd(cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	lc := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, args, clientFactory)
		},
	}
	return lc
}

func runList(_ *cobra.Command, args []string, clientFactory ClientFactory) error {
	return nil
}

func removeCmd(cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	rc := &cobra.Command{
		Use: "remove",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(cmd, args, clientFactory)
		},
	}
	return rc
}

func runRemove(_ *cobra.Command, args []string, clientFactory ClientFactory) error {
	return nil
}
