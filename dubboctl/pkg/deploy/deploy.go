// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package deploy

import (
	"errors"
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/cmd"
	"os"
	"os/exec"
	"path/filepath"
)

import (
	"github.com/AlecAivazis/survey/v2"

	"github.com/ory/viper"

	"github.com/spf13/cobra"

	"k8s.io/client-go/rest"

	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/client-go/util/homedir"
)

import (
	"github.com/apache/dubbo-kubernetes/dubboctl/internal/dubbo"
	"github.com/apache/dubbo-kubernetes/dubboctl/internal/kube"
	"github.com/apache/dubbo-kubernetes/dubboctl/internal/util"
)

const (
	basePort  = 30000
	portLimit = 32767
)

func AddDeploy(baseCmd *cobra.Command, newClient ClientFactory) {
	cmds := &cobra.Command{
		Use:   "deploy",
		Short: "Generate the k8s yaml of the application. By the way, you can choose to build the image, push the image and apply to the k8s cluster.",
		Long: `
NAME
	dubboctl deploy - Generate the k8s yaml of the application. By the way, you can choose to build the image, push the image and apply to the k8s cluster.

SYNOPSIS
	dubboctl deploy [flags]
`,
		SuggestFor: []string{"delpoy", "deplyo"},
		PreRunE: cmd.BindEnv("path", "output", "namespace", "image", "envs", "name", "containerPort",
			"targetPort", "nodePort", "apply", "useDockerfile", "force", "builder-image", "build", "context",
			"kubeConfig", "push"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeploy(cmd, newClient)
		},
	}
	cmds.Flags().StringP("namespace", "n", "default",
		"Deploy into a specific namespace")
	cmds.Flags().StringP("output", "o", "kube.yaml",
		"output kubernetes manifest")
	cmds.Flags().StringP("name", "", "",
		"The name of application")
	cmds.Flags().IntP("containerPort", "", 0,
		"The port of the deployment to listen on pod (required)")
	cmds.Flags().IntP("targetPort", "", 0,
		"The targetPort of the deployment, default to port")
	cmds.Flags().IntP("nodePort", "", 0,
		"The nodePort of the deployment to expose")

	cmds.Flags().StringP("context", "", "",
		"Context in kubeconfig to use")
	cmds.Flags().StringP("kubeConfig", "k", "",
		"Path to kubeconfig")

	cmds.Flags().StringArrayP("envs", "e", nil,
		"DeployMode variable to set in the form NAME=VALUE. "+
			"This is for the environment variables passed in by the builderpack build method.")
	cmds.Flags().StringP("builder-image", "b", "",
		"Specify a custom builder image for use by the builder other than its default.")
	cmds.Flags().BoolP("useDockerfile", "d", false,
		"Use the dockerfile with the specified path to build")
	cmds.Flags().StringP("image", "i", "",
		"Container image( [registry]/[namespace]/[name]:[tag] )")
	cmds.Flags().BoolP("push", "", true,
		"Whether to push the image to the registry center by the way")
	cmds.Flags().BoolP("force", "f", false,
		"Whether to force build")

	cmds.Flags().BoolP("build", "", true,
		"Whether to build the image")
	cmds.Flags().BoolP("apply", "a", false,
		"Whether to apply the application to the k8s cluster by the way")
	cmds.Flags().StringP("portName", "", "http",
		"Name of the port to be exposed")

	cmd.AddPathFlag(cmds)
	cmds.Flags().SetInterspersed(false)
	baseCmd.AddCommand(cmds)
}

func runDeploy(cmd *cobra.Command, newClient ClientFactory) error {
	if err := util.CreatePaths(); err != nil {
		return err
	}
	cfg := newDeployConfig(cmd)
	f, err := dubbo.NewDubbo(cfg.Path)
	if err != nil {
		return err
	}
	cfg, err = cfg.Prompt(f)
	if err != nil {
		return err
	}
	if err := cfg.Validate(cmd); err != nil {
		return err
	}

	if !f.Initialized() {
		return dubbo.NewErrNotInitialized(f.Root)
	}

	cfg.Configure(f)

	clientOptions, err := cfg.deployclientOptions()
	if err != nil {
		return err
	}
	client, done := newClient(clientOptions...)
	defer done()

	kubeEnv := true
	_, err = rest.InClusterConfig()
	if err != nil {
		kubeconfig := os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
		if len(kubeconfig) <= 0 {
			if home := homedir.HomeDir(); home != "" {
				kubeconfig = filepath.Join(home, ".kube", "config")
			}
		}
		_, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			kubeEnv = false
		}
	}

	if kubeEnv {
		err := f.CheckLabels(cfg.Namespace, client)
		if err != nil {
			return err
		}
	}

	// generate template first
	f, err = client.Deploy(cmd.Context(), f)
	if err != nil {
		return err
	}

	if cfg.Build {
		if f.Built() && !cfg.Force {
			fmt.Fprintf(cmd.OutOrStdout(), "The Application is up to date, If you still want to build, use `--force true`\n")
			return nil
		}
		if f, err = client.Build(cmd.Context(), f); err != nil {
			return err
		}
		if cfg.Push {
			if f, err = client.Push(cmd.Context(), f); err != nil {
				return err
			}
		}
	}

	if cfg.Apply {
		err := applyTok8s(cmd, f)
		if err != nil {
			return err
		}
	}

	if err = f.Write(); err != nil {
		return err
	}

	return nil
}

func (d DeployConfig) deployclientOptions() ([]dubbo.Option, error) {
	o, err := d.buildclientOptions()
	if err != nil {
		return o, err
	}
	var cliOpts []kube.CtlClientOption
	cliOpts = []kube.CtlClientOption{
		kube.WithKubeConfigPath(d.KubeConfig),
		kube.WithContext(d.Context),
	}
	cli, err := kube.NewCtlClient(cliOpts...)
	if err != nil {
		return o, err
	}
	o = append(o, dubbo.WithKubeClient(cli))
	return o, nil
}

func applyTok8s(cmd *cobra.Command, d *dubbo.Dubbo) error {
	file := filepath.Join(d.Root, d.Deploy.Output)
	c := exec.CommandContext(cmd.Context(), "kubectl", "apply", "-f", file)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Run()
	return err
}

func (c DeployConfig) Validate(cmd *cobra.Command) (err error) {
	nodePort := c.NodePort
	if nodePort != 0 && (nodePort < basePort || nodePort > portLimit) {
		return errors.New("nodePort should be between 30000 and 32767")
	}
	return nil
}

func (c *DeployConfig) Prompt(d *dubbo.Dubbo) (*DeployConfig, error) {
	var err error
	if !util.InteractiveTerminal() {
		return c, nil
	}
	buildconfig, err := c.buildConfig.Prompt(d)
	if err != nil {
		return c, err
	}
	c.buildConfig = buildconfig

	if d.Deploy.ContainerPort == 0 && c.ContainerPort == 0 {
		qs := []*survey.Question{
			{
				Name:     "containerPort",
				Validate: survey.Required,
				Prompt: &survey.Input{
					Message: "The container port",
				},
			},
		}
		if err = survey.Ask(qs, c); err != nil {
			return c, err
		}
	}
	return c, err
}

func (c DeployConfig) Configure(f *dubbo.Dubbo) {
	c.buildConfig.Configure(f)
	if c.Namespace != "" {
		f.Deploy.Namespace = c.Namespace
	}
	if c.Output != "" {
		f.Deploy.Output = c.Output
	}
	if c.ContainerPort != 0 {
		f.Deploy.ContainerPort = c.ContainerPort
	}
	if c.TargetPort != 0 {
		f.Deploy.TargetPort = c.TargetPort
	}
	if c.NodePort != 0 {
		f.Deploy.NodePort = c.NodePort
	}
	if c.PortName != "" {
		f.Deploy.PortName = c.PortName
	}
}

type DeployConfig struct {
	*buildConfig
	KubeConfig    string
	Context       string
	Build         bool
	Apply         bool
	Namespace     string
	ContainerPort int
	Output        string
	Force         bool
	TargetPort    int
	NodePort      int
	PortName      string
}

func newDeployConfig(cmd *cobra.Command) (c *DeployConfig) {
	c = &DeployConfig{
		buildConfig:   newBuildConfig(cmd),
		KubeConfig:    viper.GetString("kubeConfig"),
		Context:       viper.GetString("context"),
		Build:         viper.GetBool("build"),
		Apply:         viper.GetBool("apply"),
		Output:        viper.GetString("output"),
		Namespace:     viper.GetString("namespace"),
		Force:         viper.GetBool("force"),
		ContainerPort: viper.GetInt("containerPort"),
		TargetPort:    viper.GetInt("targetPort"),
		NodePort:      viper.GetInt("nodePort"),
		PortName:      viper.GetString("portName"),
	}
	return
}
