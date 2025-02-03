package cmd

import (
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk/dubbo"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/util"
	"github.com/ory/viper"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"path/filepath"
)

type buildConfig struct {
	Build bool
	Path  string
	Push  bool
}

func (c buildConfig) buildclientOptions() ([]sdk.Option, error) {
	var do []sdk.Option
	return do, nil
}

type pushConfig struct {
	Push bool
	Path string
}

type applyConfig struct {
	Apply bool
	Path  string
}

func newBuildConfig(cmd *cobra.Command) *buildConfig {
	bc := &buildConfig{
		Build: viper.GetBool("build"),
		Path:  viper.GetString("path"),
	}
	return bc
}

func newPushConfig(cmd *cobra.Command) *pushConfig {
	pc := &pushConfig{
		Push: viper.GetBool("push"),
		Path: viper.GetString("path"),
	}
	return pc
}

func newApplyConfig(cmd *cobra.Command) *applyConfig {
	ac := &applyConfig{
		Apply: viper.GetBool("apply"),
		Path:  viper.GetString("path"),
	}
	return ac
}

func ImageCmd(ctx cli.Context, cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	ipc := imagePushCmd(cmd, clientFactory)
	iac := imageApplyCmd(cmd, clientFactory)

	ic := &cobra.Command{
		Use:   "image",
		Short: "Used to build and push images, apply to cluster",
	}
	ic.AddCommand(ipc)
	ic.AddCommand(iac)
	return ic
}

func imagePushCmd(cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	pc := &cobra.Command{
		Use:     "push",
		Short:   "Build and push to images",
		Long:    "The push subcommand used to push images",
		Example: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPush(cmd, args, clientFactory)
		},
	}
	return pc
}

func runPush(cmd *cobra.Command, args []string, clientFactory ClientFactory) error {
	if err := util.GetCreatePath(); err != nil {
		return err
	}
	config := newBuildConfig(cmd)

	fp, err := dubbo.NewDubboConfig(config.Path)
	if err != nil {
		return err
	}

	if !fp.Initialized() {
	}

	clientOptions, err := config.buildclientOptions()
	if err != nil {
		return err
	}
	client, done := clientFactory(clientOptions...)
	defer done()
	if fp, err = client.Build(cmd.Context(), fp); err != nil {
		return err
	}
	if config.Push {
		if fp, err = client.Push(cmd.Context(), fp); err != nil {
			return err
		}
	}

	err = fp.WriteYamlFile()
	if err != nil {
		return err
	}

	return nil
}

func imageApplyCmd(cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	ac := &cobra.Command{
		Use:     "apply",
		Short:   "Deploy the images to the cluster",
		Long:    "The apply subcommand used to apply images",
		Example: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runApply(cmd, args, clientFactory)
		},
	}
	return ac
}

func runApply(cmd *cobra.Command, args []string, clientFactory ClientFactory) error {
	if err := util.GetCreatePath(); err != nil {
		return err
	}

	config := newApplyConfig(cmd)

	fp, err := dubbo.NewDubboConfig(config.Path)
	if err != nil {
		return err
	}

	if !fp.Initialized() {
	}
	if err := applyToCluster(cmd, fp); err != nil {
		return err
	}

	return nil
}

func applyToCluster(cmd *cobra.Command, dc *dubbo.DubboConfig) error {
	file := filepath.Join(dc.Root)
	ec := exec.CommandContext(cmd.Context(), "kubectl", "apply", "-f", file)
	ec.Stdout = os.Stdout
	ec.Stderr = os.Stderr
	if err := ec.Run(); err != nil {
		return err
	}
	return nil
}
