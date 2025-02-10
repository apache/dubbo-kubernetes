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
	deployment bool
}

func addHubFlags(cmd *cobra.Command, iArgs *imageArgs) {
	cmd.PersistentFlags().BoolVarP(&iArgs.dockerfile, "file", "f", false, "Specify the file as a dockerfile")
}

func addDeployFlags(cmd *cobra.Command, iArgs *imageArgs) {
	cmd.PersistentFlags().BoolVarP(&iArgs.deployment, "apply", "a", false, "Deploy to Cluster")
}

type hubConfig struct {
	Dockerfile   bool
	Image        string
	BuilderImage string
	Path         string
}

type deployConfig struct {
	Path string
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
	}
	return hc
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

func imageHubCmd(cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	iArgs := &imageArgs{}
	hc := &cobra.Command{
		Use:     "hub",
		Short:   "Build and Push to images",
		Long:    "The hub subcommand used to build and push images",
		Example: "",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("accepts %d arg(s), received %d", args, len(args))
			}

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
		PreRunE: bindEnv("file"),
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
	config := newHubConfig(cmd)

	fp, err := dubbo.NewDubboConfig(config.Path)
	if err != nil {
		return err
	}

	config, err = config.prompt(fp)
	if err != nil {
		return err
	}

	if !fp.Initialized() {
		return util.NewErrNotInitialized(fp.Root)
	}

	config.configure(fp)

	clientOptions, err := config.imageClientOptions()
	if err != nil {
		return err
	}

	client, done := clientFactory(clientOptions...)
	defer done()

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
		PreRunE: bindEnv("apply"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runApply(cmd, args, clientFactory)
		},
	}
	addDeployFlags(hc, iArgs)
	return hc
}

func runApply(cmd *cobra.Command, args []string, clientFactory ClientFactory) error {
	dc := &dubbo.DubboConfig{}
	file := filepath.Join(dc.Root)
	ec := exec.CommandContext(cmd.Context(), "kubectl", "apply", "-f", file)
	ec.Stdout = os.Stdout
	ec.Stderr = os.Stderr
	if err := ec.Run(); err != nil {
		return err
	}
	return nil
}

func (hc *hubConfig) configure(dc *dubbo.DubboConfig) {
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

func (hc *hubConfig) prompt(dc *dubbo.DubboConfig) (*hubConfig, error) {
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
