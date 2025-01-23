package cmd

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/util"
	"github.com/apache/dubbo-kubernetes/operator/cmd/cluster"
	"github.com/spf13/cobra"
)

type repoArgs struct{}

func addRepoFlags(cmd *cobra.Command, rArgs *repoArgs) {}

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
	}

	cluster.AddFlags(rc, rootArgs)
	addRepoFlags(rc, rArgs)
	rc.AddCommand(ad)
	rc.AddCommand(li)
	rc.AddCommand(re)
	return rc
}

func addCmd(cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	ac := &cobra.Command{
		Use:   "add [name] [URL]",
		Short: "Add a new template library.",
		Long:  "The add subcommand is used to add a new template library.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdd(cmd, args, clientFactory)
		},
	}
	return ac
}

func runAdd(cmd *cobra.Command, args []string, clientFactory ClientFactory) (err error) {
	if err = util.CreatePath(); err != nil {
		return
	}
	client, done := clientFactory()
	defer done()

	if len(args) != 2 {
		return fmt.Errorf("Usage: dubboctl repo add [name] [URL]")
	}

	p := struct {
		name string
		url  string
	}{}
	if len(args) > 0 {
		p.name = args[0]
	}
	if len(args) > 1 {
		p.url = args[1]
	}

	var n string
	if n, err = client.Repositories().Add(p.name, p.url); err != nil {
		return
	}

	fmt.Printf("%s Repositories added.\n", n)
	return
}

func listCmd(cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	lc := &cobra.Command{
		Use:     "list",
		Short:   "View the list of template library.",
		Long:    "The list subcommand is used to view the repositories that have been added.",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, args, clientFactory)
		},
	}
	return lc
}

func runList(cmd *cobra.Command, args []string, clientFactory ClientFactory) (err error) {
	client, done := clientFactory()
	defer done()

	list, err := client.Repositories().All()
	if err != nil {
		return
	}

	for _, l := range list {
		fmt.Println(l.Name + "\t" + l.URL())
	}
	return
}

func removeCmd(cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	rc := &cobra.Command{
		Use:     "remove [name]",
		Short:   "Remove an existing template library.",
		Long:    "The delete subcommand is used to delete a template from an existing repository.",
		Aliases: []string{"delete"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(cmd, args, clientFactory)
		},
	}
	return rc
}

func runRemove(cmd *cobra.Command, args []string, clientFactory ClientFactory) (err error) {
	client, done := clientFactory()
	defer done()

	p := struct {
		name string
		sure bool
	}{}
	if len(args) > 0 {
		p.name = args[0]
	}
	p.sure = true

	if err = client.Repositories().Remove(p.name); err != nil {
		return
	}

	fmt.Printf("%s Repositories removed.\n", p.name)
	return
}
