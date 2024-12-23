package cluster

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/spf13/cobra"
	"strings"
)

type ManifestGenerateArgs struct {
	files        []string
	sets         []string
	manifestPath string
}

func (a *ManifestGenerateArgs) String() string {
	var b strings.Builder
	b.WriteString("files:   " + fmt.Sprint(a.files) + "\n")
	b.WriteString("sets:           " + fmt.Sprint(a.sets) + "\n")
	b.WriteString("manifestPath: " + a.manifestPath + "\n")
	return b.String()
}

func addManifestGenerateFlags(cmd *cobra.Command, args *ManifestGenerateArgs) {
	cmd.PersistentFlags().StringSliceVarP(&args.files, "filename", "f", nil, "")
	cmd.PersistentFlags().StringArrayVarP(&args.sets, "set", "s", nil, "")
	cmd.PersistentFlags().StringVarP(&args.manifestPath, "manifests", "d", "", "")
}

func ManifestCmd(ctx cli.Context) *cobra.Command {
	rootArgs := &RootArgs{}
	mgcArgs := &ManifestGenerateArgs{}
	mgc := ManifestGenerateCmd(ctx, rootArgs, mgcArgs)
	ic := InstallCmd(ctx)
	mc := &cobra.Command{
		Use:   "manifest",
		Short: "dubbo manifest related commands",
		Long:  "The manifest command will generates and diffs dubbo manifests.",
	}
	addFlags(mc, rootArgs)
	addFlags(mgc, rootArgs)
	addManifestGenerateFlags(mgc, mgcArgs)
	mc.AddCommand(mgc)
	mc.AddCommand(ic)
	return mc
}

func ManifestGenerateCmd(ctx cli.Context, _ *RootArgs, mgArgs *ManifestGenerateArgs) *cobra.Command {
	return &cobra.Command{
		Use:   "generate",
		Short: "Generates an Dubbo install manifest",
		Long:  "The generate subcommand generates an Dubbo install manifest and outputs to the console by default.",
		Example: `  # Generate a default Istio installation
  istioctl manifest generate

  # Generate the demo profile
  istioctl manifest generate --set profile=demo
`,
		Args: nil,
		RunE: nil,
	}
}
