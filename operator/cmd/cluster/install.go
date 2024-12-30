package cluster

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/apache/dubbo-kubernetes/operator/pkg/install"
	"github.com/apache/dubbo-kubernetes/operator/pkg/render"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util/clog"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util/clog/log"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util/progress"
	"github.com/apache/dubbo-kubernetes/pkg/art"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/util/pointer"
	"github.com/spf13/cobra"
	"io"
	"os"
	"strings"
	"time"
)

var installerScope = log.RegisterScope("installer")

type installArgs struct {
	files            []string
	sets             []string
	manifestPath     string
	skipConfirmation bool
	waitTimeout      time.Duration
}

func (i *installArgs) String() string {
	var b strings.Builder
	b.WriteString("files:    " + (fmt.Sprint(i.files) + "\n"))
	b.WriteString("sets:    " + (fmt.Sprint(i.sets) + "\n"))
	b.WriteString("waitTimeout: " + fmt.Sprint(i.waitTimeout) + "\n")
	return b.String()
}

func addInstallFlags(cmd *cobra.Command, args *installArgs) {
	cmd.PersistentFlags().StringSliceVarP(&args.files, "files", "f", nil, `Path to the file containing the dubboOperator's custom resources`)
	cmd.PersistentFlags().StringArrayVarP(&args.sets, "set", "s", nil, `Override dubboOperator values, such as selecting profiles, etc`)
	cmd.PersistentFlags().DurationVar(&args.waitTimeout, "wait-timeout", 300*time.Second, "Maximum time to wait for Dubbo resources in each component to be ready")
}

func InstallCmd(ctx cli.Context) *cobra.Command {
	return InstallCmdWithArgs(ctx, &RootArgs{}, &installArgs{})
}

func InstallCmdWithArgs(ctx cli.Context, rootArgs *RootArgs, iArgs *installArgs) *cobra.Command {
	ic := &cobra.Command{
		Use:   "install",
		Short: "Applies an Dubbo manifest, installing or reconfiguring Dubbo on a cluster",
		Long:  "The install command generates an Dubbo install manifest and applies it to a cluster",
		Example: ` # Apply a default dubboctl installation.
  dubboctl install
  
  # Apply a config file.
  dubboctl install -f dop.yaml
  
  # Apply a default profile.
  dubboctl install --profile=demo
		`,
		Aliases: []string{"apply"},
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			kubeClient, err := ctx.CLIClient()
			if err != nil {
				return err
			}
			p := NewPrinterForWriter(cmd.OutOrStderr())
			cl := clog.NewConsoleLogger(cmd.OutOrStdout(), cmd.ErrOrStderr(), installerScope)
			p.Printf("%v\n", art.DubboColoredArt())
			return Install(kubeClient, rootArgs, iArgs, cl, cmd.OutOrStdout(), p)
		},
	}
	addFlags(ic, rootArgs)
	addInstallFlags(ic, iArgs)
	return ic
}

func Install(kubeClient kube.CLIClient, rootArgs *RootArgs, iArgs *installArgs, cl clog.Logger, stdOut io.Writer, p Printer) error {
	setFlags := applyFlagAliases(iArgs.sets, iArgs.manifestPath)
	manifests, vals, err := render.GenerateManifest(iArgs.files, setFlags, cl, kubeClient)
	if err != nil {
		return fmt.Errorf("generate config: %v", err)
	}
	profile := pointer.NonEmptyOrDefault(vals.GetPathString("spec.profile"), "default")
	if !rootArgs.DryRun && !iArgs.skipConfirmation {
		prompt := fmt.Sprintf("You are currently selecting the %q profile to install into the cluster. Do you want to proceed? (y/N)", profile)
		if !OptionDeterminate(prompt, stdOut) {
			p.Println("Canceled Completed.")
			os.Exit(1)
		}
	}
	i := install.Installer{
		DryRun:       rootArgs.DryRun,
		SkipWait:     false,
		Kube:         kubeClient,
		Values:       vals,
		WaitTimeout:  iArgs.waitTimeout,
		ProgressInfo: progress.NewInfo(),
		Logger:       cl,
	}
	if err := i.InstallManifests(manifests); err != nil {
		return fmt.Errorf("failed to install manifests: %v", err)
	}
	return nil
}

func applyFlagAliases(flags []string, manifestsPath string) []string {
	if manifestsPath != "" {
		flags = append(flags, fmt.Sprintf("manifestsPath=%s", manifestsPath))
	}
	return flags
}
