package cluster

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/apache/dubbo-kubernetes/operator/pkg/render"
	"github.com/apache/dubbo-kubernetes/operator/pkg/uninstall"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util/clog"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util/progress"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/spf13/cobra"
)

type uninstallArgs struct {
	files        string
	sets         []string
	manifestPath string
	purge        bool
}

func addUninstallFlags(cmd *cobra.Command, args *uninstallArgs) {
	cmd.PersistentFlags().StringVarP(&args.files, "filename", "f", "",
		"The filename of the IstioOperator CR.")
	cmd.PersistentFlags().StringArrayVarP(&args.sets, "set", "s", nil, "Override dubboOperator values, such as selecting profiles, etc")
	cmd.PersistentFlags().BoolVar(&args.purge, "purge", false, "Remove all dubbo-related source code")
}

func UninstallCmd(ctx cli.Context) *cobra.Command {
	rootArgs := &RootArgs{}
	uiArgs := &uninstallArgs{}
	uicmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Dubbo-related resources",
		Long:  "Uninstalling Dubbo from the Cluster",
		Example: `·# Uninstall a single control plane by dop file
  dubboctl uninstall -f dop.yaml
  
  # Uninstall all control planes and shared resources
  dubboctl uninstall --purge`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return UnInstall(cmd, ctx, rootArgs, uiArgs)
		},
	}
	addFlags(uicmd, rootArgs)
	addUninstallFlags(uicmd, uiArgs)
	return uicmd
}

func UnInstall(cmd *cobra.Command, ctx cli.Context, rootArgs *RootArgs, uiArgs *uninstallArgs) error {
	cl := clog.NewConsoleLogger(cmd.OutOrStdout(), cmd.ErrOrStderr(), installerScope)
	var kubeClient kube.CLIClient
	var err error
	if err != nil {
		return err
	}
	pl := progress.NewInfo()
	setFlags := applyFlagAliases(uiArgs.sets, uiArgs.manifestPath)
	files := []string{}
	if uiArgs.files != "" {
		files = append(files, uiArgs.files)
	}
	vals, err := render.MergeInputs(files, setFlags)
	if err != nil {
		return err
	}
	objectsList, err := uninstall.GetPrunedResources(
		vals.GetPathString("metadata.name"),
		vals.GetPathString("metadata.namespace"),
	)
	if err != nil {
		return err
	}
	if err := uninstall.DeleteObjectsList(kubeClient, rootArgs.DryRun, cl, objectsList); err != nil {
		return fmt.Errorf("failed to delete control plane resources by revision: %v", err)
	}
	pl.SetState(progress.StateUninstallComplete)
	return nil
}
