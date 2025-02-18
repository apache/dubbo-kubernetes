package cmd

import (
	"errors"
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk/dubbo"
	"github.com/apache/dubbo-kubernetes/operator/cmd/cluster"
	"github.com/ory/viper"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

type createArgs struct {
	dirname  string
	language string
	template string
}

func addCreateFlags(cmd *cobra.Command, tempArgs *createArgs) {
	cmd.PersistentFlags().StringVarP(&tempArgs.language, "language", "l", "", "java or go language")
	cmd.PersistentFlags().StringVarP(&tempArgs.template, "template", "t", "", "java or go sdk template")
	cmd.PersistentFlags().StringVar(&tempArgs.dirname, "dirname", "", "java or go sdk template custom directory name")
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
  dubboctl create sdk --language java --template common --dirname mydubbo

  dubboctl create sdk -l java -t common --dirname mydubbo
  
  dubboctl create sdk -l java -t common --dirname myrepo/mydubbo

  # Create a go sample sdk.
  dubboctl create sdk --language go --template common --dirname mydubbogo

  dubboctl create sdk -l go -t common --dirname mydubbogo

  dubboctl create sdk -l go -t common --dirname myrepo/mydubbogo
`,
		PreRunE: bindEnv("language", "template", "dirname"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd, args, clientFactory)
		},
	}
}

type createConfig struct {
	Path       string
	Runtime    string
	Template   string
	Repo       string
	DirName    string
	Initialzed bool
}

func newCreateConfig(cmd *cobra.Command, args []string, clientFactory ClientFactory) (dcfg createConfig, err error) {
	var absolutePath string
	absolutePath = cwd()

	dcfg = createConfig{
		DirName:    viper.GetString("dirname"),
		Path:       absolutePath + "/" + viper.GetString("dirname"),
		Runtime:    viper.GetString("language"),
		Template:   viper.GetString("template"),
		Initialzed: viper.GetBool("initialzed"),
	}
	fmt.Printf("Name:     %v\n", dcfg.DirName)
	fmt.Printf("Path:     %v\n", dcfg.Path)
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

	if err = dcfg.validate(dclient); err != nil {
		return err
	}

	_, err = dclient.Initialize(&dubbo.DubboConfig{
		Root:     dcfg.Path,
		Name:     dcfg.DirName,
		Runtime:  dcfg.Runtime,
		Template: dcfg.Template,
	}, dcfg.Initialzed, cmd)
	if err != nil {
		return err
	}
	fmt.Printf("Custom Dubbo %v sdk was successfully created.\n", dcfg.Runtime)
	return nil
}

type ErrNoRuntime error
type ErrInvalidRuntime error
type ErrInvalidTemplate error

func (c createConfig) validate(client *sdk.Client) (err error) {
	if c.Runtime == "" {
		return noRuntimeError(client)
	}

	if c.Runtime != "" && c.Repo == "" &&
		!isValidRuntime(client, c.Runtime) {
		return newInvalidRuntimeError(client, c.Runtime)
	}

	if c.Template != "" && c.Repo == "" &&
		!isValidTemplate(client, c.Runtime, c.Template) {
		return newInvalidTemplateError(client, c.Runtime, c.Template)
	}

	return
}

func newInvalidRuntimeError(client *sdk.Client, runtime string) error {
	b := strings.Builder{}
	fmt.Fprintf(&b, "The language runtime '%v' is not recognized.\n", runtime)
	runtimes, err := client.Runtimes()
	if err != nil {
		return err
	}
	for _, v := range runtimes {
		fmt.Fprintf(&b, "  %v\n", v)
	}
	return ErrInvalidRuntime(errors.New(b.String()))
}

func isValidTemplate(client *sdk.Client, runtime, template string) bool {
	if !isValidRuntime(client, runtime) {
		return false
	}
	templates, err := client.Templates().List(runtime)
	if err != nil {
		return false
	}
	for _, v := range templates {
		if v == template {
			return true
		}
	}
	return false
}

func isValidRuntime(client *sdk.Client, runtime string) bool {
	runtimes, err := client.Runtimes()
	if err != nil {
		return false
	}
	for _, v := range runtimes {
		if v == runtime {
			return true
		}
	}
	return false
}

func noRuntimeError(client *sdk.Client) error {
	b := strings.Builder{}
	fmt.Fprintln(&b, "Required flag \"language\" not set.")
	fmt.Fprintln(&b, "Available language runtimes are:")
	runtimes, err := client.Runtimes()
	if err != nil {
		return err
	}
	for _, v := range runtimes {
		fmt.Fprintf(&b, "  %v\n", v)
	}
	return ErrNoRuntime(errors.New(b.String()))
}

func newInvalidTemplateError(client *sdk.Client, runtime, template string) error {
	b := strings.Builder{}
	fmt.Fprintf(&b, "The template '%v' was not found for language runtime '%v'.\n", template, runtime)
	fmt.Fprintln(&b, "Available templates for this language runtime are:")
	templates, err := client.Templates().List(runtime)
	if err != nil {
		return err
	}
	for _, v := range templates {
		fmt.Fprintf(&b, "  %v\n", v)
	}
	return ErrInvalidTemplate(errors.New(b.String()))
}

func cwd() (cwd string) {
	cwd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("Unable to determine current working directory: %v", err))
	}
	return cwd
}
