//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package component

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/operator/pkg/apis"
	"github.com/apache/dubbo-kubernetes/operator/pkg/values"
)

type Name string

const (
	BaseComponentName           Name = "Base"
	DubboDiscoveryComponentName Name = "Dubbod"
)

type Component struct {
	// UserFacingName is the component name in user-facing cases.
	UserFacingName Name
	// ContainerName maps a Name to the name of the container in a Deployment.
	ContainerName string
	// SpecName is the yaml key in the DubboOperator spec.
	SpecName string
	// ResourceType maps a Name to the type of the rendered k8s resource.
	ResourceType string
	// ResourceName maps a Name to the name of the rendered k8s resource.
	ResourceName string
	// Default defines whether the component is enabled by default.
	Default bool
	// HelmSubDir is a mapping between a component name and the subdirectory of the component Chart.
	HelmSubDir string
	// HelmValuesTreeRoot is the tree root in values YAML files for the component.
	HelmValuesTreeRoot string
}

var AllComponents = []Component{
	{
		UserFacingName:     BaseComponentName,
		SpecName:           "base",
		ResourceType:       "Base",
		HelmSubDir:         "base",
		Default:            true,
		HelmValuesTreeRoot: "global",
	},
	{
		UserFacingName: DubboDiscoveryComponentName,
		SpecName:       "dubbo",
		ResourceType:   "Deployment",
		ResourceName:   "dubbo",
		ContainerName:  "dubbo-discovery",
		Default:        true,
		HelmSubDir:     "dubbo-control/dubbo-discovery",
	},
}

var (
	userFacingCompNames = map[Name]string{
		BaseComponentName:           "Dubbo Resource Core",
		DubboDiscoveryComponentName: "Dubbo Control Plane",
	}

	Icons = map[Name]string{
		BaseComponentName:           "⚡️",
		DubboDiscoveryComponentName: "⚡️",
	}
)

func (c Component) Get(merged values.Map) ([]apis.DefaultCompSpec, error) {
	defaultNamespace := merged.GetPathString("metadata.namespace")
	var defaultResp []apis.DefaultCompSpec
	buildSpec := func(m values.Map) (apis.DefaultCompSpec, error) {
		spec, err := values.ConvertMap[apis.DefaultCompSpec](m)
		if err != nil {
			return apis.DefaultCompSpec{}, fmt.Errorf("fail to convert %v: %v", c.SpecName, err)
		}
		if spec.Namespace == "" {
			spec.Namespace = defaultNamespace
		}
		if spec.Namespace == "" {
			spec.Namespace = "dubbo-system"
		}
		spec.Raw = m
		return spec, nil
	}
	// Single component
	s, ok := merged.GetPathMap("spec.components." + c.SpecName)
	if !ok {
		return defaultResp, nil
	}
	spec, err := buildSpec(s)
	if err != nil {
		return nil, err
	}
	if !(spec.Enabled.GetValueOrTrue()) {
		return nil, nil
	}
	return []apis.DefaultCompSpec{spec}, nil
}

// UserFacingCompName returns the name of the given component that should be displayed to the user in high
// level CLIs (like progress log).
func UserFacingCompName(name Name) string {
	s, ok := userFacingCompNames[name]
	if !ok {
		return "Unknown"
	}
	return s
}
