/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dubbo

import (
	"context"
	_ "embed"
	"os"
	"strings"
	template2 "text/template"

	"github.com/apache/dubbo-kubernetes/app/dubboctl/internal/util"
)

const (
	deployTemplateFile = "deploy.tpl"
)

//go:embed deploy.tpl
var deployTemplate string

type DeployApp struct{}

type DeployerOpt func(deployer *DeployApp)

func NewDeployer(opts ...DeployerOpt) *DeployApp {
	d := &DeployApp{}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

type Deployment struct {
	Labels          []Label
	Envs            []Env
	Name            string
	Namespace       string
	Image           string
	Secret          string
	Replicas        int
	Revisions       int
	Port            int
	TargetPort      int
	NodePort        int
	UseNodePort     bool
	RequestCpu      int
	RequestMem      int
	LimitCpu        int
	LimitMem        int
	MinReplicas     int
	MaxReplicas     int
	ServiceAccount  string
	ImagePullPolicy string
	UseProm         bool
	UsePromScrape   bool
	PromPath        string
	PromPort        int
	UseSkywalking   bool
}

func (d *DeployApp) Deploy(ctx context.Context, f *Dubbo, option ...DeployOption) (DeploymentResult, error) {
	ns := f.Deploy.Namespace

	var nodePort int
	var err error
	if f.Deploy.NodePort == 0 {
		nodePort = 0
	}
	nodePort = f.Deploy.NodePort

	text, err := util.LoadTemplate("", deployTemplateFile, deployTemplate)
	if err != nil {
		return DeploymentResult{
			Status:    Failed,
			Namespace: ns,
		}, err
	}

	targetPort := f.Deploy.TargetPort
	if targetPort == 0 {
		targetPort = f.Deploy.ContainerPort
	}

	var envs []Env
	for _, pair := range f.Deploy.Envs {
		parts := strings.Split(pair, "=")

		if len(parts) == 2 {
			key := &parts[0]
			value := &parts[1]
			env := Env{
				Name:  key,
				Value: value,
			}
			envs = append(envs, env)
		}
	}

	var labels []Label
	for _, pair := range f.Deploy.Labels {
		parts := strings.Split(pair, "=")

		if len(parts) == 2 {
			key := &parts[0]
			value := &parts[1]
			label := Label{
				Key:   key,
				Value: value,
			}
			labels = append(labels, label)
		}
	}

	path := f.Root + "/" + f.Deploy.Output
	out, err := os.Create(path)
	if err != nil {
		return DeploymentResult{
			Status:    Deployed,
			Namespace: ns,
		}, err
	}

	promPath := f.Deploy.PromPath
	promPort := f.Deploy.PromPort
	t := template2.Must(template2.New("deployTemplate").Parse(text))
	err = t.Execute(out, Deployment{
		Name:            f.Name,
		Namespace:       ns,
		Image:           f.Image,
		Secret:          f.Deploy.Secret,
		Replicas:        f.Deploy.Replicas,
		Revisions:       f.Deploy.Revisions,
		Port:            f.Deploy.ContainerPort,
		TargetPort:      targetPort,
		NodePort:        f.Deploy.NodePort,
		UseNodePort:     nodePort > 0,
		RequestCpu:      f.Deploy.RequestCpu,
		RequestMem:      f.Deploy.RequestMem,
		LimitCpu:        f.Deploy.LimitCpu,
		LimitMem:        f.Deploy.LimitMem,
		MinReplicas:     f.Deploy.MinReplicas,
		MaxReplicas:     f.Deploy.MaxReplicas,
		ServiceAccount:  f.Deploy.ServiceAccount,
		ImagePullPolicy: f.Deploy.ImagePullPolicy,
		UseProm:         f.Deploy.PromPath != "" || f.Deploy.PromPort > 0,
		UsePromScrape:   true,
		PromPath:        promPath,
		PromPort:        promPort,
		UseSkywalking:   f.Deploy.UseSkywalking,
		Envs:            envs,
		Labels:          labels,
	})
	if err != nil {
		return DeploymentResult{
			Status:    Deployed,
			Namespace: ns,
		}, err
	}

	return DeploymentResult{
		Status:    Deployed,
		Namespace: ns,
	}, nil
}
