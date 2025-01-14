package cmd

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk/dubbo"
	"github.com/apache/dubbo-kubernetes/operator/cmd/cluster"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
)

type templateArgs struct {
	template string
}

func addTemplateFlags(cmd *cobra.Command, tempArgs *templateArgs) {
	cmd.PersistentFlags().StringVarP(&tempArgs.template, "template", "t", "", "java or go sdk template")
}

func CreateCmd(_ cli.Context, cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	rootArgs := &cluster.RootArgs{}
	tempArgs := &templateArgs{}
	sc := sdkGenerateCmd(cmd, clientFactory)
	cc := &cobra.Command{
		Use:   "create",
		Short: "Create a custom dubbo sdk sample",
		Long:  "The create command will generates dubbo sdk.",
	}
	cluster.AddFlags(cc, rootArgs)
	cluster.AddFlags(sc, rootArgs)
	addTemplateFlags(cc, tempArgs)
	cc.AddCommand(sc)
	return cc
}

func sdkGenerateCmd(_ *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "sdk",
		Short: "Generate sdk samples for Dubbo supported languages",
		Long:  "The sdk subcommand generates an sdk sample provided by Dubbo supported languages.",
		Example: `  # Create a java sample sdk.
  dubboctl create sdk java -n mydubbo

  # Create a go sample sdk.
  dubboctl create sdk go -n mydubbogo
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd, args, clientFactory)
		},
	}
}

type createConfig struct {
	Path       string
	Template   string
	Name       string
	Initialzed bool
}

func newCreateConfig(_ *cobra.Command, args []string, _ ClientFactory) (cfg createConfig, err error) {
	var (
		path         string
		dirName      string
		absolutePath string
	)

	if len(args) >= 1 {
		path = args[0]
	}

	dirName, absolutePath = deriveNameAndAbsolutePathFromPath(path)

	cfg = createConfig{
		Name: dirName,
		Path: absolutePath,
	}
	fmt.Printf("Path:         %v\n", cfg.Path)
	fmt.Printf("Template:     %v\n", cfg.Template)
	return
}

func runCreate(cmd *cobra.Command, args []string, clientFactory ClientFactory) error {
	dcfg, err := newCreateConfig(cmd, args, clientFactory)
	if err != nil {
		return err
	}

	dclient, cancel := clientFactory()
	defer cancel()

	_, err = dclient.Initialize(&dubbo.DubboConfig{
		Root:     dcfg.Path,
		Name:     dcfg.Name,
		Template: dcfg.Template,
	}, dcfg.Initialzed, cmd)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStderr(), "Created %v dubbo sdk in %v\n", args[0], dcfg.Path)
	return nil
}

func cwd() (cwd string) {
	cwd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("Unable to determine current working directory: %v", err))
	}
	return cwd
}

func deriveNameAndAbsolutePathFromPath(path string) (string, string) {
	var absPath string

	if path == "" {
		path = cwd()
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", ""
	}

	pathParts := strings.Split(strings.TrimRight(path, string(os.PathSeparator)), string(os.PathSeparator))
	return pathParts[len(pathParts)-1], absPath
}
