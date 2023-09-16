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
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/apache/dubbo-kubernetes/app/dubboctl/internal/kube"
	corev1 "k8s.io/api/core/v1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	client2 "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/dubbo-kubernetes/app/dubboctl/internal/dubbo"
	"github.com/apache/dubbo-kubernetes/app/dubboctl/internal/util"

	"github.com/AlecAivazis/survey/v2"
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
		PreRunE: bindEnv("path", "output", "namespace", "image", "envs", "name", "secret", "replicas",
			"revisions", "containerPort", "targetPort", "nodePort", "requestCpu", "limitMem",
			"minReplicas", "serviceAccount", "serviceAccount", "imagePullPolicy", "maxReplicas", "limitCpu", "requestMem",
			"apply", "useDockerfile", "force", "builder-image", "nobuild", "context", "kubeConfig", "push"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeploy(cmd, newClient)
		},
	}

	cmd.Flags().BoolP("nobuild", "", false,
		"Skip the step of building the image.")
	cmd.Flags().BoolP("apply", "a", false,
		"Whether to apply the application to the k8s cluster by the way")
	cmd.Flags().StringP("output", "o", "kube.yaml",
		"output kubernetes manifest")
	cmd.Flags().StringP("namespace", "n", "dubbo-system",
		"Deploy into a specific namespace")
	cmd.Flags().StringArrayP("envs", "e", nil,
		"Environment variable to set in the form NAME=VALUE. "+
			"This is for the environment variables passed in by the builderpack build method.")
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
	cmd.Flags().StringP("builder-image", "b", "",
		"Specify a custom builder image for use by the builder other than its default.")
	cmd.Flags().BoolP("useDockerfile", "d", false,
		"Use the dockerfile with the specified path to build")
	cmd.Flags().StringP("image", "i", "",
		"Container image( [registry]/[namespace]/[name]:[tag] )")
	cmd.Flags().BoolP("push", "", true,
		"Whether to push the image to the registry center by the way")
	cmd.Flags().BoolP("force", "f", false,
		"Whether to force push")
	cmd.Flags().StringP("context", "", "",
		"Context in kubeconfig to use")
	cmd.Flags().StringP("kubeConfig", "k", "",
		"Path to kubeconfig")

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

	cfg.Configure(f)

	clientOptions, err := cfg.deployclientOptions()
	if err != nil {
		return err
	}
	client, done := newClient(clientOptions...)
	defer done()

	key := client2.ObjectKey{
		Namespace: metav1.NamespaceSystem,
		Name:      cfg.Namespace,
	}

	err = client.KubeCtl.Get(context.Background(), key, &corev1.Namespace{})
	if err != nil {
		if errors2.IsNotFound(err) {
			nsObj := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: metav1.NamespaceSystem,
					Name:      cfg.Namespace,
				},
			}
			if err := client.KubeCtl.Create(context.Background(), nsObj); err != nil {
				return err
			}
			return nil
		} else {
			return fmt.Errorf("failed to check if namespace %v exists: %v", cfg.Namespace, err)
		}
	}

	namespaceSelector := client2.MatchingLabels{
		"dubbo-deploy": "enabled",
	}
	nsList := &corev1.NamespaceList{}
	if err = client.KubeCtl.List(context.Background(), nsList, namespaceSelector); err != nil {
		return err
	}
	var namespace string
	if len(nsList.Items) > 0 {
		namespace = nsList.Items[0].Name
	}

	env := os.Getenv("DUBBO_DEPLOY_NS")
	if env != "" {
		namespace = env
	}

	if namespace != "" {
		zkSelector := client2.MatchingLabels{
			"dubbo.apache.org/zookeeper": "true",
		}

		zkList := &corev1.ServiceList{}
		if err := client.KubeCtl.List(context.Background(), zkList, zkSelector, client2.InNamespace(namespace)); err != nil {
			return err
		}
		var name string
		var dns string
		if len(zkList.Items) > 0 {
			name = zkList.Items[0].Name
			dns = fmt.Sprintf("%s.%s.svc", name, namespace)
			f.Deploy.ZookeeperAddress = dns
		}

		nacosSelector := client2.MatchingLabels{
			"dubbo.apache.org/nacos": "true",
		}

		nacosList := &corev1.ServiceList{}
		if err := client.KubeCtl.List(context.Background(), nacosList, nacosSelector, client2.InNamespace(namespace)); err != nil {
			return err
		}
		if len(nacosList.Items) > 0 {
			name = nacosList.Items[0].Name
			dns = fmt.Sprintf("%s.%s.svc", name, namespace)
			f.Deploy.NacosAddress = dns
		}

		promSelector := client2.MatchingLabels{
			"dubbo.apache.org/prometheus": "true",
		}

		promList := &corev1.ServiceList{}
		if err := client.KubeCtl.List(context.Background(), promList, promSelector, client2.InNamespace(namespace)); err != nil {
			return err
		}
		if len(promList.Items) > 0 {
			f.Deploy.UseProm = true
		}
	}

	if !cfg.NoBuild {
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
	f, err = client.Deploy(cmd.Context(), f)
	if err != nil {
		return err
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

	return f.Stamp()
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
}

type DeployConfig struct {
	*buildConfig
	KubeConfig      string
	Context         string
	NoBuild         bool
	Apply           bool
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
}

func newDeployConfig(cmd *cobra.Command) (c *DeployConfig) {
	c = &DeployConfig{
		buildConfig:     newBuildConfig(cmd),
		KubeConfig:      viper.GetString("kubeConfig"),
		Context:         viper.GetString("context"),
		NoBuild:         viper.GetBool("nobuild"),
		Apply:           viper.GetBool("apply"),
		Output:          viper.GetString("output"),
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
	}
	return
}
