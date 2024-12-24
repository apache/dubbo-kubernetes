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
	"os"
)

const (
	AllResourcesRemovedWarning                  = "All Dubbo resources will be pruned from the cluster\n"
	PurgeWithRevisionOrOperatorSpecifiedWarning = "Purge uninstall will remove all Dubbo resources, ignoring the specified revision or operator file"
)

type uninstallArgs struct {
	files            string
	sets             []string
	manifestPath     string
	purge            bool
	skipConfirmation bool
}

func addUninstallFlags(cmd *cobra.Command, args *uninstallArgs) {
	cmd.PersistentFlags().StringVarP(&args.files, "filename", "f", "",
		"The filename of the DubboOperator CR.")
	cmd.PersistentFlags().StringArrayVarP(&args.sets, "set", "s", nil, `Override dubboOperator values, such as selecting profiles, etc.`)
	cmd.PersistentFlags().BoolVar(&args.purge, "purge", false, `Remove all dubbo related source code.`)
	cmd.PersistentFlags().BoolVarP(&args.skipConfirmation, "skip-confirmation", "y", false, `The skipConfirmation determines whether the user is prompted for confirmation.`)
}

func UninstallCmd(ctx cli.Context) *cobra.Command {
	rootArgs := &RootArgs{}
	uiArgs := &uninstallArgs{}
	uicmd := &cobra.Command{
		Use:          "uninstall",
		Short:        "Uninstall Dubbo related resources",
		Long:         "The uninstall command will uninstall the dubbo cluster",
		SilenceUsage: true,
		Example: ` # Uninstall a single control plane by dop file
  dubboctl uninstall -f dop.yaml
  
  # Uninstall all control planes and shared resources
  dubboctl uninstall --purge`,
		Args: func(cmd *cobra.Command, args []string) error {
			if uiArgs.files == "" && !uiArgs.purge {
				return fmt.Errorf("at least one of the --filename or --purge flags must be set")
			}
			if len(args) > 0 {
				return fmt.Errorf("dubboctl uninstall does not take arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return Uninstall(cmd, ctx, rootArgs, uiArgs)
		},
	}
	addFlags(uicmd, rootArgs)
	addUninstallFlags(uicmd, uiArgs)
	return uicmd
}

func Uninstall(cmd *cobra.Command, ctx cli.Context, rootArgs *RootArgs, uiArgs *uninstallArgs) error {
	cl := clog.NewConsoleLogger(cmd.OutOrStdout(), cmd.ErrOrStderr(), installerScope)
	var kubeClient kube.CLIClient
	var err error
	kubeClient, err = ctx.CLIClientWithRevision("")
	if err != nil {
		return err
	}

	pl := progress.NewInfo()
	if uiArgs.purge && uiArgs.files != "" {
		cl.LogAndPrint(PurgeWithRevisionOrOperatorSpecifiedWarning)
	}

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
		kubeClient,
		vals.GetPathString("metadata.name"),
		vals.GetPathString("metadata.namespace"),
		uiArgs.purge,
	)
	if err != nil {
		return err
	}

	preCheck(cmd, uiArgs, cl, rootArgs.DryRun)

	if err := uninstall.DeleteObjectsList(kubeClient, rootArgs.DryRun, cl, objectsList); err != nil {
		return fmt.Errorf("failed to delete control plane resources by revision: %v", err)
	}

	pl.SetState(progress.StateUninstallComplete)

	return nil
}

func preCheck(cmd *cobra.Command, uiArgs *uninstallArgs, cl *clog.ConsoleLogger, dryRun bool) {
	needConfirmation, message := false, ""
	if uiArgs.purge {
		needConfirmation = true
		message += AllResourcesRemovedWarning
	}
	if dryRun || uiArgs.skipConfirmation {
		cl.LogAndPrint(message)
		return
	}
	message += "Proceed? (y/N)"
	if needConfirmation && !OptionDeterminate(message, cmd.OutOrStdout()) {
		cmd.Print("Canceled Completed.\n")
		os.Exit(1)
	}
}
