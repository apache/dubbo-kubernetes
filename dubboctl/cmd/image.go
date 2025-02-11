package cmd

import (
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/hub/builder/dockerfile"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/hub/builder/pack"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk/dubbo"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/util"
	"github.com/ory/viper"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type imageArgs struct {
	dockerfile bool
	builder    bool
	output     string
}

func addHubFlags(cmd *cobra.Command, iArgs *imageArgs) {
	cmd.PersistentFlags().BoolVarP(&iArgs.dockerfile, "file", "f", false, "Specify the file as a dockerfile")
	cmd.PersistentFlags().BoolVarP(&iArgs.builder, "builder", "b", false, "The builder generates the image")
}

func addDeployFlags(cmd *cobra.Command, iArgs *imageArgs) {
	cmd.PersistentFlags().StringVarP(&iArgs.output, "output", "o", "dubbo-deploy.yaml", "The output generates k8s yaml file")

}

type hubConfig struct {
	Dockerfile   bool
	Builder      bool
	Image        string
	BuilderImage string
	Path         string
}

type deployConfig struct {
	*hubConfig
	Output    string
	Namespace string
	Port      int
	Path      string
}

func ImageCmd(ctx cli.Context, cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	ihc := imageHubCmd(cmd, clientFactory)
	idc := imageDeployCmd(cmd, clientFactory)
	ic := &cobra.Command{
		Use:   "image",
		Short: "Used to build and push images, apply to cluster",
	}
	ic.AddCommand(ihc)
	ic.AddCommand(idc)
	return ic
}

func newHubConfig(cmd *cobra.Command) *hubConfig {
	hc := &hubConfig{
		Dockerfile: viper.GetBool("file"),
		Builder:    viper.GetBool("builder"),
	}
	return hc
}

func newDeployConfig(cmd *cobra.Command) *deployConfig {
	dc := &deployConfig{
		hubConfig: newHubConfig(cmd),
		Output:    viper.GetString("output"),
	}
	return dc
}

func (hc hubConfig) imageClientOptions() ([]sdk.Option, error) {
	var do []sdk.Option
	if hc.Dockerfile {
		do = append(do, sdk.WithBuilder(dockerfile.NewBuilder()))
	} else {
		do = append(do, sdk.WithBuilder(pack.NewBuilder()))
	}
	return do, nil
}

func (dc deployConfig) deployClientOptions() ([]sdk.Option, error) {
	i, err := dc.imageClientOptions()
	if err != nil {
		return i, err
	}

	return i, nil
}

func imageHubCmd(cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	iArgs := &imageArgs{}
	hc := &cobra.Command{
		Use:     "hub",
		Short:   "Build and Push to images",
		Long:    "The hub subcommand used to build and push images",
		Example: "",
		Args: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("file") {
				if len(args) != 1 {
					return fmt.Errorf("you must provide exactly one argument when using the -f flag: the path to the Dockerfile")
				}

				df := args[0]
				if !strings.HasSuffix(df, "Dockerfile") {
					return fmt.Errorf("the provided file must be a Dockerfile when using the -f flag")
				}
			}
			return nil
		},
		PreRunE: bindEnv("file", "builder"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHub(cmd, args, clientFactory)
		},
	}
	addHubFlags(hc, iArgs)
	return hc
}

func runHub(cmd *cobra.Command, args []string, clientFactory ClientFactory) error {
	if err := util.GetCreatePath(); err != nil {
		return err
	}
	hcfg := newHubConfig(cmd)
	fp, err := dubbo.NewDubboConfig(hcfg.Path)
	if err != nil {
		return err
	}

	hcfg, err = hcfg.hubPrompt(fp)
	if err != nil {
		return err
	}

	if !fp.Initialized() {
		return util.NewErrNotInitialized(fp.Root)
	}

	hcfg.checkHubConfig(fp)

	clientOptions, err := hcfg.imageClientOptions()
	if err != nil {
		return err
	}

	client, done := clientFactory(clientOptions...)
	defer done()

	if fp, err = client.Deploy(cmd.Context(), fp); err != nil {
		return err
	}
	if fp.Built() {
		return nil
	}
	if fp, err = client.Build(cmd.Context(), fp); err != nil {
		return err
	}

	if fp, err = client.Push(cmd.Context(), fp); err != nil {
		return err
	}

	err = fp.WriteFile()
	if err != nil {
		return err
	}

	return nil
}

func imageDeployCmd(cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	iArgs := &imageArgs{}
	hc := &cobra.Command{
		Use:     "deploy",
		Short:   "Deploy to cluster",
		Long:    "The deploy subcommand used to deploy to cluster",
		Example: "",
		PreRunE: bindEnv("output"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeploy(cmd, args, clientFactory)
		},
	}
	addDeployFlags(hc, iArgs)
	return hc
}

func runDeploy(cmd *cobra.Command, args []string, clientFactory ClientFactory) error {
	if err := util.GetCreatePath(); err != nil {
		return err
	}
	dcfg := newDeployConfig(cmd)

	fp, err := dubbo.NewDubboConfig(dcfg.Path)
	if err != nil {
		return err
	}

	dcfg, err = dcfg.deployPrompt(fp)
	if err != nil {
		return err
	}

	dcfg.checkDeployConfig(fp)

	clientOptions, err := dcfg.deployClientOptions()
	if err != nil {
		return err
	}

	client, done := clientFactory(clientOptions...)
	defer done()
	if fp, err = client.Deploy(cmd.Context(), fp); err != nil {
		return err
	}

	if err := apply(cmd, fp); err != nil {
		return err
	}

	err = fp.WriteFile()
	if err != nil {
		return err
	}

	return nil
}

func apply(cmd *cobra.Command, dc *dubbo.DubboConfig) error {
	file := filepath.Join(dc.Root, dc.Deploy.Output)
	ec := exec.CommandContext(cmd.Context(), "kubectl", "apply", "-f", file)
	ec.Stdout = os.Stdout
	ec.Stderr = os.Stderr
	if err := ec.Run(); err != nil {
		return err
	}
	return nil
}

func (hc *hubConfig) checkHubConfig(dc *dubbo.DubboConfig) {
	if hc.Path == "" {
		root, err := os.Getwd()
		if err != nil {
			return
		}
		dc.Root = root
	} else {
		dc.Root = hc.Path
	}
	if hc.BuilderImage != "" {
		dc.Build.BuilderImages["pack"] = hc.BuilderImage
	}
	if hc.Image != "" {
		dc.Image = hc.Image
	}
}

func (dc deployConfig) checkDeployConfig(dc2 *dubbo.DubboConfig) {
	dc.checkHubConfig(dc2)
	if dc.Output != "" {
		dc2.Deploy.Output = dc.Output
	}
	if dc.Namespace != "" {
		dc2.Deploy.Namespace = dc.Namespace
	}
	if dc.Port != 0 {
		dc2.Deploy.Port = dc.Port
	}
}

func (hc *hubConfig) hubPrompt(dc *dubbo.DubboConfig) (*hubConfig, error) {
	var err error
	if !util.InteractiveTerminal() {
		return hc, nil
	}

	if hc.Image == "" && dc.Image == "" {
		qs := []*survey.Question{
			{
				Name:     "image",
				Validate: survey.Required,
				Prompt: &survey.Input{
					Message: "Please enter the image tag ([REGISTRY]/[USERNAME]/[IMAGENAME]:tag)\n  Image: ",
					Default: hc.Image,
				},
			},
		}
		if err = survey.Ask(qs, hc); err != nil {
			return hc, err
		}
	}
	return hc, err
}

func (dc *deployConfig) deployPrompt(dc2 *dubbo.DubboConfig) (*deployConfig, error) {
	var err error
	if !util.InteractiveTerminal() {
		return dc, nil
	}
	if dc2.Deploy.Namespace == "" {
		qs := []*survey.Question{
			{
				Name:     "namespace",
				Validate: survey.Required,
				Prompt: &survey.Input{
					Message: "Namespace",
				},
			},
		}
		if err = survey.Ask(qs, dc); err != nil {
			return dc, err
		}
	}

	buildconfig, err := dc.hubConfig.hubPrompt(dc2)
	if err != nil {
		return dc, err
	}

	dc.hubConfig = buildconfig

	if dc2.Deploy.Port == 0 && dc.Port == 0 {
		qs := []*survey.Question{
			{
				Name:     "port",
				Validate: survey.Required,
				Prompt: &survey.Input{
					Message: "Port",
				},
			},
		}
		if err = survey.Ask(qs, dc); err != nil {
			return dc, err
		}
	}

	return dc, err
}
