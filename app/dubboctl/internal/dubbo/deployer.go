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
	"fmt"
	"os"
	template2 "text/template"
)

import (
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
	Name        string
	Namespace   string
	Image       string
	Port        int
	TargetPort  int
	NodePort    int
	UseNodePort bool
	UseProm     bool
}

func (d *DeployApp) Deploy(ctx context.Context, f *Dubbo, option ...DeployOption) (DeploymentResult, error) {
	ns := f.Deploy.Namespace

	var nodePort int
	var err error
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

	path := f.Root + "/" + f.Deploy.Output
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "WARNING! The file already exists in this directory and has been overwritten.")
	}

	out, err := os.Create(path)
	if err != nil {
		return DeploymentResult{
			Status:    Failed,
			Namespace: ns,
		}, err
	}

	t := template2.Must(template2.New("deployTemplate").Parse(text))
	err = t.Execute(out, Deployment{
		Name:        f.Name,
		Namespace:   ns,
		Image:       f.Image,
		Port:        f.Deploy.ContainerPort,
		TargetPort:  targetPort,
		NodePort:    f.Deploy.NodePort,
		UseNodePort: nodePort > 0,
		UseProm:     f.Deploy.UseProm,
	})
	if err != nil {
		return DeploymentResult{
			Status:    Failed,
			Namespace: ns,
		}, err
	}

	return DeploymentResult{
		Status:    Deployed,
		Namespace: ns,
	}, nil
}
