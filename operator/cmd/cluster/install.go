package cluster

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/apache/dubbo-kubernetes/operator/pkg/install"
	"github.com/apache/dubbo-kubernetes/operator/pkg/render"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util/clog"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util/clog/log"
	"github.com/apache/dubbo-kubernetes/pkg/art"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/util/pointer"
	"github.com/spf13/cobra"
	"io"
	"os"
	"strings"
	"time"
)

var installerScope = log.RegisterScope("installer", "installer")

type installArgs struct {
	Files            []string
	Sets             []string
	Revision         string
	ManifestPath     string
	SkipConfirmation bool
	ReadinessTimeout time.Duration
}

func (i *installArgs) String() string {
	var b strings.Builder
	b.WriteString("Files:    " + (fmt.Sprint(i.Files) + "\n"))
	b.WriteString("Sets:    " + (fmt.Sprint(i.Sets) + "\n"))
	b.WriteString("Revision:    " + (fmt.Sprint(i.Revision) + "\n"))
	b.WriteString("ManifestPath:    " + (fmt.Sprint(i.ManifestPath) + "\n"))
	return b.String()
}

func InstallCmd(ctx cli.Context) *cobra.Command {
	return InstallCmdWithArgs(ctx, &RootArgs{}, &installArgs{})
}

func InstallCmdWithArgs(ctx cli.Context, rootArgs *RootArgs, iArgs *installArgs) *cobra.Command {
	ic := &cobra.Command{
		Use:   "install",
		Short: "Applies an Dubbo manifest, installing or reconfiguring Dubbo on a cluster",
		Long:  "The install command generates an Dubbo install manifest and applies it to a cluster",
		Example: `
        # Apply a default dubboctl installation.
		dubboctl install
		# Apply a default profile.
		dubboctl install --profile=default
		# Apply a config file
		dubboctl install -f my-config.yaml
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
			p.Printf("%v\n", art.DubboArt())
			return Install(kubeClient, rootArgs, iArgs, cl, cmd.OutOrStdout(), p)
		},
	}
	return ic
}

func Install(kubeClient kube.CLIClient, rootArgs *RootArgs, iArgs *installArgs, cl clog.Logger, stdOut io.Writer, p Printer) error {
	setFlags := applyFlagAliases(iArgs.Sets, iArgs.ManifestPath, iArgs.Revision)
	manifests, vals, err := render.GenerateManifest(iArgs.Files, setFlags, cl, kubeClient)
	if err != nil {
		return fmt.Errorf("generate config: %v", err)
	}
	profile := pointer.NonEmptyOrDefault(vals.GetPathString("spec.profile"), "default")
	if !rootArgs.DryRun && !iArgs.SkipConfirmation {
		prompt := fmt.Sprintf("You are currently selecting the %q profile to install into the cluster. %v Do you want to proceed? (y/N)", profile, vals)
		if !OptionDeterminator(prompt, stdOut) {
			p.Println("Cancelled.")
			os.Exit(1)
		}
	}
	i := install.Installer{
		DryRun:   rootArgs.DryRun,
		SkipWait: false,
		Kube:     kubeClient,
		Values:   vals,
	}
	if err := i.InstallManifests(manifests); err != nil {
		return fmt.Errorf("failed to install manifests: %v", err)
	}
	return nil
}

func applyFlagAliases(flags []string, manifestsPath string, revision string) []string {
	if manifestsPath != "" {
		flags = append(flags, fmt.Sprintf("manifestsPath=%s", manifestsPath))
	}
	return flags
}
