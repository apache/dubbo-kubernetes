package cmd

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk/dubbo"
	"github.com/ory/viper"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"path/filepath"
)

type BuildConfig struct {
	Build bool
}

type PushConfig struct {
	Push bool
}

type ApplyConfig struct {
	Apply bool
}

func newBuildConfig(cmd *cobra.Command) *BuildConfig {
	bc := &BuildConfig{
		Build: viper.GetBool("build"),
	}
	return bc
}

func newPushConfig(cmd *cobra.Command) *PushConfig {
	pc := &PushConfig{
		Push: viper.GetBool("push"),
	}
	return pc
}

func newApplyConfig(cmd *cobra.Command) *ApplyConfig {
	ac := &ApplyConfig{
		Apply: viper.GetBool("apply"),
	}
	return ac
}

func ImageCmd(ctx cli.Context, cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	ibc := imageBuildCmd(cmd, clientFactory)
	ipc := imagePushCmd(cmd, clientFactory)
	iac := imageApplyCmd(cmd, clientFactory)

	ic := &cobra.Command{
		Use:   "image",
		Short: "Used to build images, push images, apply to cluster",
	}
	ic.AddCommand(ibc)
	ic.AddCommand(ipc)
	ic.AddCommand(iac)
	return ic
}

func imageBuildCmd(cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	bc := &cobra.Command{
		Use:     "build",
		Short:   "build to images",
		Long:    "The build subcommand used to build images",
		Example: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBuild(cmd, args, clientFactory)
		},
	}
	return bc
}

func runBuild(cmd *cobra.Command, args []string, clientFactory ClientFactory) error {
	return fmt.Errorf("TODO")
}

func imagePushCmd(cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	pc := &cobra.Command{
		Use:     "push",
		Short:   "push to images",
		Long:    "The push subcommand used to push images",
		Example: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPush(cmd, args, clientFactory)
		},
	}
	return pc
}

func runPush(cmd *cobra.Command, args []string, clientFactory ClientFactory) error {
	return fmt.Errorf("TODO")
}

func imageApplyCmd(cmd *cobra.Command, clientFactory ClientFactory) *cobra.Command {
	ac := &cobra.Command{
		Use:     "apply",
		Short:   "apply to images",
		Long:    "The apply subcommand used to apply images",
		Example: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runApply(cmd, args, clientFactory)
		},
	}
	return ac
}

func runApply(cmd *cobra.Command, args []string, clientFactory ClientFactory) error {
	if err := applyToCluster(cmd, nil); err != nil {
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
