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

type createArgs struct {
	language string
	template string
}

func addCreateFlags(cmd *cobra.Command, tempArgs *createArgs) {
	cmd.PersistentFlags().StringVarP(&tempArgs.language, "language", "l", "", "java or go language")
	cmd.PersistentFlags().StringVarP(&tempArgs.template, "template", "t", "", "java or go sdk template")
}

type bindFunc func(*cobra.Command, []string) error

func bindEnv(flags ...string) bindFunc {
	return func(cmd *cobra.Command, args []string) (err error) {
		for _, flag := range flags {
			if err = viper.BindPFlag(flag, cmd.Flags().Lookup(flag)); err != nil {
				return
			}
		}
		viper.AutomaticEnv()
		return
	}
}

func CreateCmd(_ cli.Context, cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	rootArgs := &cluster.RootArgs{}
	tempArgs := &createArgs{}
	sc := sdkGenerateCmd(cmd, clientFactory)
	cc := &cobra.Command{
		Use:   "create",
		Short: "Create a custom dubbo sdk sample",
		Long:  "The create command will generates dubbo sdk.",
	}
	cluster.AddFlags(cc, rootArgs)
	cluster.AddFlags(sc, rootArgs)
	addCreateFlags(sc, tempArgs)
	cc.AddCommand(sc)
	return cc
}

func sdkGenerateCmd(cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "sdk",
		Short: "Generate sdk samples for Dubbo supported languages",
		Long:  "The sdk subcommand generates an sdk sample provided by Dubbo supported languages.",
		Example: `  # Create a java sample sdk.
  dubboctl create sdk --language java -t common mydubbo

  # Create a go sample sdk.
  dubboctl create sdk --language go -t common mydubbogo
`,
		PreRunE: bindEnv("language", "template"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd, args, clientFactory)
		},
	}
}

type createConfig struct {
	Path       string
	Runtime    string
	Template   string
	Name       string
	Initialzed bool
}

func newCreateConfig(cmd *cobra.Command, args []string, clientFactory ClientFactory) (dcfg createConfig, err error) {
	var (
		path         string
		dirName      string
		absolutePath string
	)

	if len(args) >= 1 {
		path = args[0]
	}

	dirName, absolutePath = deriveNameAndAbsolutePathFromPath(path)
	dcfg = createConfig{
		Name:       dirName,
		Path:       absolutePath,
		Runtime:    viper.GetString("language"),
		Template:   viper.GetString("template"),
		Initialzed: viper.GetBool("initialzed"),
	}
	fmt.Printf("Name:     %v\n", dirName)
	fmt.Printf("Path:     %v\n", absolutePath)
	fmt.Printf("Language:     %v\n", dcfg.Runtime)
	fmt.Printf("Template:     %v\n", dcfg.Template)
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
		Runtime:  dcfg.Runtime,
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
