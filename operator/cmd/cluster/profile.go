package cluster

import (
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/apache/dubbo-kubernetes/operator/pkg/helm"
	"github.com/spf13/cobra"
	"sort"
)

type profileListArgs struct {
	manifestsPath string
}

func addProfileListFlags(cmd *cobra.Command, args *profileListArgs) {
	cmd.PersistentFlags().StringVarP(&args.manifestsPath, "manifests", "d", "", "Specify a path to a directory of charts and profiles")
}

func ProfileCmd(ctx cli.Context) *cobra.Command {
	rootArgs := &RootArgs{}
	plArgs := &profileListArgs{}
	plc := profileListCmd(rootArgs, plArgs)
	pc := &cobra.Command{
		Use:   "profile",
		Short: "Commands related to Dubbo configuration profiles",
		Long:  "The profile command lists, dumps or diffs Dubbo configuration profiles.",
		Example: "dubboctl profile list\n" +
			"dubboctl install --set profile=demo  # Use a profile from the list",
	}
	AddFlags(pc, rootArgs)
	pc.AddCommand(plc)
	return pc
}

func profileListCmd(rootArgs *RootArgs, plArgs *profileListArgs) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Lists available Dubbo configuration profiles",
		Long:  "The list subcommand lists the available Dubbo configuration profiles.",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return profileList(cmd, rootArgs, plArgs)
		},
	}
}

func profileList(cmd *cobra.Command, args *RootArgs, plArgs *profileListArgs) error {
	profiles, err := helm.ListProfiles(plArgs.manifestsPath)
	if err != nil {
		return err
	}

	if len(profiles) == 0 {
		cmd.Println("No profiles available.")
	} else {
		cmd.Println("Dubbo configuration profiles:")
		sort.Strings(profiles)
		for _, profile := range profiles {
			cmd.Printf("    %s\n", profile)
		}
	}

	return nil
}
