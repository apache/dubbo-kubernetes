package cmd

import (
	"github.com/AlecAivazis/survey/v2"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/hub/builder/pack"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk/dubbo"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/util"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"path/filepath"
)

type hubConfig struct {
	Image        string
	BuilderImage string
	Path         string
}

func ImageCmd(ctx cli.Context, cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	ihc := imageHubCmd(cmd, clientFactory)
	ic := &cobra.Command{
		Use:   "image",
		Short: "Used to build and push images, apply to cluster",
	}
	ic.AddCommand(ihc)
	return ic
}

func newHubConfig(cmd *cobra.Command) *hubConfig {
	hc := &hubConfig{}
	return hc
}

func (c hubConfig) imageClientOptions() ([]sdk.Option, error) {
	var do []sdk.Option
	do = append(do, sdk.WithBuilder(pack.NewBuilder()))
	return do, nil
}

func imageHubCmd(cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	bc := &cobra.Command{
		Use:     "hub",
		Short:   "Build and Push to images",
		Long:    "The hub subcommand used to build and push images",
		Example: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHub(cmd, args, clientFactory)
		},
	}
	return bc
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

func runApply(cmd *cobra.Command, dc *dubbo.DubboConfig) error {
	file := filepath.Join(dc.Root)
	ec := exec.CommandContext(cmd.Context(), "kubectl", "apply", "-f", file)
	ec.Stdout = os.Stdout
	ec.Stderr = os.Stderr
	if err := ec.Run(); err != nil {
		return err
	}
	return nil
}

func (c *hubConfig) configure(dc *dubbo.DubboConfig) {
	if c.Path == "" {
		root, err := os.Getwd()
		if err != nil {
			return
		}
		dc.Root = root
	} else {
		dc.Root = c.Path
	}
	if c.BuilderImage != "" {
		dc.Build.BuilderImages["pack"] = c.BuilderImage
	}
	if c.Image != "" {
		dc.Image = c.Image
	}
}

func (c *hubConfig) prompt(dc *dubbo.DubboConfig) (*hubConfig, error) {
	var err error
	if !util.InteractiveTerminal() {
		return c, nil
	}

	if c.Image == "" && dc.Image == "" {
		qs := []*survey.Question{
			{
				Name:     "image",
				Validate: survey.Required,
				Prompt: &survey.Input{
					Message: "Please enter the image tag ([REGISTRY]/[USERNAME]/[IMAGENAME]:tag)\n Image: ",
					Default: c.Image,
				},
			},
		}
		if err = survey.Ask(qs, c); err != nil {
			return c, err
		}
	}
	return c, err
}
