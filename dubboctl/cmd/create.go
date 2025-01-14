package cmd

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk/dubbo"
	"github.com/apache/dubbo-kubernetes/operator/cmd/cluster"
	"github.com/ory/viper"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
)

type templateArgs struct {
	template string
}

func addTemplateFlags(cmd *cobra.Command, args *templateArgs) {
	cmd.PersistentFlags().StringVarP(&args.template, "template", "t", "", "java or go sdk template")
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

func sdkGenerateCmd(cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "sdk",
		Short: "Generate sdk samples for Dubbo supported languages",
		Long:  "The sdk subcommand generates an SDK sample provided by Dubbo supported languages.",
		Example: `  # Create a java sample sdk.
  dubboctl create sdk java -t mydubbo

  # Create a go sample sdk.
  dubboctl create sdk go -t mydubbogo
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd, clientFactory)
		},
	}
}

type createArgs struct {
	Path       string
	Runtime    string
	Template   string
	Name       string
	Initialzed bool
}

func runCreate(cmd *cobra.Command, clientFactory ClientFactory) error {
	dcfg, err := newCreate(cmd, clientFactory)
	if err != nil {
		return err
	}
	client, cancel := clientFactory()
	defer cancel()
	_, err = client.Initialize(&dubbo.DubboConfig{
		Root:     dcfg.Path,
		Name:     dcfg.Name,
		Runtime:  dcfg.Runtime,
		Template: dcfg.Template,
	}, dcfg.Initialzed, cmd)
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStderr(), "Created %v dubbo sdk in %v\n", dcfg.Runtime, dcfg.Path)
	return nil
}

func newCreate(cmd *cobra.Command, clientFactory ClientFactory) (dcfg createArgs, err error) {
	var (
		path         string
		dirName      string
		absolutePath string
	)
	dirName, absolutePath = deriveNameAndAbsolutePathFromPath(path)

	dcfg = createArgs{
		Path:     absolutePath,
		Runtime:  viper.GetString("language"),
		Template: viper.GetString("template"),
		Name:     dirName,
	}

	fmt.Printf("Path:         %v\n", dcfg.Path)
	fmt.Printf("Language:     %v\n", dcfg.Runtime)
	fmt.Printf("Template:     %v\n", dcfg.Template)

	return createArgs{}, nil
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
