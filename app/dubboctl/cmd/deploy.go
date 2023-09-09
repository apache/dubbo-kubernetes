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

package cmd

import (
	"errors"
	"fmt"

	"github.com/apache/dubbo-kubernetes/app/dubboctl/internal/util"

	"github.com/AlecAivazis/survey/v2"
	"github.com/apache/dubbo-kubernetes/app/dubboctl/internal/dubbo"
	"github.com/ory/viper"

	"github.com/spf13/cobra"
)

const (
	basePort  = 30000
	portLimit = 32767
)

func addDeploy(baseCmd *cobra.Command, newClient ClientFactory) {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Generate deploy manifests or apply it by the way",
		Long: `
NAME
	dubboctl deploy - Deploy an application

SYNOPSIS
	dubboctl deploy [flags]
`,
		SuggestFor: []string{"delpoy", "deplyo"},
		PreRunE: bindEnv("path", "output", "namespace", "image", "env", "labels",
			"name", "secret", "replicas", "revisions", "containerPort", "targetPort", "nodePort", "requestCpu", "limitMem",
			"minReplicas", "serviceAccount", "serviceAccount", "imagePullPolicy", "usePromScrape", "promPath",
			"promPort", "useSkywalking", "maxReplicas", "limitCpu", "requestMem"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeploy(cmd, newClient)
		},
	}

	cmd.Flags().StringP("output", "o", "kube.yaml",
		"output kubernetes manifest")
	cmd.Flags().StringP("namespace", "n", "dubbo-system",
		"Deploy into a specific namespace")
	cmd.Flags().StringP("image", "i", "",
		"Full image name in the form [registry]/[namespace]/[name]:[tag]@[digest]")
	cmd.Flags().StringArrayP("env", "e", nil,
		"Environment variable to set in the form NAME=VALUE. ")
	cmd.Flags().StringArrayP("labels", "l", nil,
		"Labels variable to set in the form KEY=VALUE. ")
	cmd.Flags().StringP("name", "", "",
		"The name of deployment (required)")
	cmd.Flags().StringP("secret", "", "",
		"The secret to image pull from registry")
	cmd.Flags().IntP("replicas", "", 3,
		"The number of replicas to deploy")
	cmd.Flags().IntP("revisions", "", 5,
		"The number of replicas to deploy")
	cmd.Flags().IntP("containerPort", "", 0,
		"The port of the deployment to listen on pod (required)")
	cmd.Flags().IntP("targetPort", "", 0,
		"The targetPort of the deployment, default to port")
	cmd.Flags().IntP("nodePort", "", 0,
		"The nodePort of the deployment to expose")
	cmd.Flags().IntP("requestCpu", "", 500,
		"The request cpu to deploy")
	cmd.Flags().IntP("requestMem", "", 512,
		"The request memory to deploy")
	cmd.Flags().IntP("limitCpu", "", 1000,
		"The limit cpu to deploy")
	cmd.Flags().IntP("limitMem", "", 1024,
		"The limit memory to deploy")
	cmd.Flags().IntP("minReplicas", "", 3,
		"The min replicas to deploy")
	cmd.Flags().IntP("maxReplicas", "", 10,
		"The max replicas to deploy")
	cmd.Flags().StringP("serviceAccount", "", "",
		"TheServiceAccount for the deployment")
	cmd.Flags().StringP("imagePullPolicy", "", "",
		"The image pull policy of the deployment, default to IfNotPresent")
	cmd.Flags().BoolP("usePromScrape", "", false,
		"use promScrape or not")
	cmd.Flags().StringP("promPath", "", "",
		"prometheus path")
	cmd.Flags().IntP("promPort", "", 0,
		"prometheus port")
	cmd.Flags().BoolP("useSkywalking", "", false,
		"use SkyWalking or not")

	addPathFlag(cmd)
	baseCmd.AddCommand(cmd)
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
	if err != nil {
		return err
	}

	if !f.Initialized() {
		return dubbo.NewErrNotInitialized(f.Root)
	}
	f, err = cfg.Configure(f)
	if err != nil {
		return err
	}

	client, done := newClient()
	defer done()

	f, err = client.Deploy(cmd.Context(), f)
	if err != nil {
		return err
	}

	return f.Write()
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

	if d.Image == "" && c.Image == "" {
		qs := []*survey.Question{
			{
				Name:     "image",
				Validate: survey.Required,
				Prompt: &survey.Input{
					Message: "The container image( [registry]/[namespace]/[name]:[tag] ). For example: docker.io/sjmshsh/testapp:latest",
					Default: c.Image,
				},
			},
		}
		if err = survey.Ask(qs, c); err != nil {
			return c, err
		}
	}
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

func (c DeployConfig) Configure(f *dubbo.Dubbo) (*dubbo.Dubbo, error) {
	if c.Path == "" {
		f.Root = "."
	} else {
		f.Root = c.Path
	}
	if c.Image != "" {
		f.Image = c.Image
	}
	if c.Labels != nil {
		f.Deploy.Labels = c.Labels
	}
	if c.Envs != nil {
		f.Deploy.Envs = c.Envs
	}
	if c.Namespace != "" {
		f.Deploy.Namespace = c.Namespace
	}
	if c.Secret != "" {
		f.Deploy.Secret = c.Secret
	}
	if c.Output != "" {
		f.Deploy.Output = c.Output
	}
	if c.Replicas != 0 {
		f.Deploy.Replicas = c.Replicas
	}
	if c.Revisions != 0 {
		f.Deploy.Revisions = c.Revisions
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
	if c.RequestCpu != 0 {
		f.Deploy.RequestCpu = c.RequestCpu
	}
	if c.RequestMem != 0 {
		f.Deploy.RequestMem = c.RequestMem
	}
	if c.LimitCpu != 0 {
		f.Deploy.LimitCpu = c.LimitCpu
	}
	if c.LimitMem != 0 {
		f.Deploy.LimitMem = c.LimitMem
	}
	if c.MinReplicas != 0 {
		f.Deploy.MinReplicas = c.MinReplicas
	}
	if c.MaxReplicas != 0 {
		f.Deploy.MaxReplicas = c.MaxReplicas
	}
	if c.ServiceAccount != "" {
		f.Deploy.ServiceAccount = c.ServiceAccount
	}
	if c.ImagePullPolicy != "" {
		f.Deploy.ImagePullPolicy = c.ImagePullPolicy
	}
	f.Deploy.UsePromScrape = c.UsePromScrape
	if c.PromPath != "" {
		f.Deploy.PromPath = c.PromPath
	}
	if c.PromPort != 0 {
		f.Deploy.PromPort = c.PromPort
	}

	if c.UseSkywalking != false || (c.UseSkywalking == false && f.Deploy.UseSkywalking == true) {
		f.Deploy.UseSkywalking = c.UseSkywalking
	}
	return f, nil
}

type DeployConfig struct {
	Path            string
	Image           string
	Labels          []string
	Envs            []string
	Namespace       string
	Secret          string
	Replicas        int
	Revisions       int
	ContainerPort   int
	Output          string
	Force           bool
	TargetPort      int
	NodePort        int
	RequestCpu      int
	RequestMem      int
	LimitCpu        int
	LimitMem        int
	MinReplicas     int
	MaxReplicas     int
	ServiceAccount  string
	ImagePullPolicy string
	UsePromScrape   bool
	PromPath        string
	PromPort        int
	UseSkywalking   bool
}

func newDeployConfig(cmd *cobra.Command) (c *DeployConfig) {
	c = &DeployConfig{
		Path:            viper.GetString("path"),
		Image:           viper.GetString("image"),
		Output:          viper.GetString("output"),
		Labels:          viper.GetStringSlice("labels"),
		Envs:            viper.GetStringSlice("env"),
		Namespace:       viper.GetString("namespace"),
		Secret:          viper.GetString("secret"),
		Replicas:        viper.GetInt("replicas"),
		Force:           viper.GetBool("force"),
		Revisions:       viper.GetInt("revisions"),
		ContainerPort:   viper.GetInt("containerPort"),
		TargetPort:      viper.GetInt("targetPort"),
		NodePort:        viper.GetInt("nodePort"),
		RequestCpu:      viper.GetInt("requestCpu"),
		RequestMem:      viper.GetInt("requestMem"),
		LimitCpu:        viper.GetInt("limitCpu"),
		LimitMem:        viper.GetInt("limitMem"),
		MinReplicas:     viper.GetInt("minReplicas"),
		MaxReplicas:     viper.GetInt("maxReplicas"),
		ServiceAccount:  viper.GetString("serviceAccount"),
		ImagePullPolicy: viper.GetString("imagePullPolicy"),
		UsePromScrape:   viper.GetBool("usePromScrape"),
		PromPath:        viper.GetString("promPath"),
		PromPort:        viper.GetInt("promPort"),
		UseSkywalking:   viper.GetBool("useSkywalking"),
	}
	// NOTE: .Env should be viper.GetStringSlice, but this returns unparsed
	// results and appears to be an open issue since 2017:
	// https://github.com/spf13/viper/issues/380
	var err error
	if c.Envs, err = cmd.Flags().GetStringArray("env"); err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "error reading envs: %v", err)
	}
	if c.Labels, err = cmd.Flags().GetStringArray("labels"); err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "error reading labels: %v", err)
	}
	return
}
