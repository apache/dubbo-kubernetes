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

package config

import (
	"fmt"
	"os"

	"github.com/apache/dubbo-kubernetes/pkg/bootstrap"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	meshv1alpha1 "github.com/kdubbo/api/mesh/v1alpha1"
)

// ConstructProxyConfig returns proxyConfig
func ConstructProxyConfig(meshGlobalSetupFile, proxyConfigEnv string) (*meshv1alpha1.ProxyConfig, error) {
	annotations, err := bootstrap.ReadPodAnnotations("")
	if err != nil {
		if os.IsNotExist(err) {
			log.Debugf("failed to read pod annotations: %v", err)
		} else {
			log.Warnf("failed to read pod annotations: %v", err)
		}
	}
	var fileMeshContents string
	if fileExists(meshGlobalSetupFile) {
		contents, err := os.ReadFile(meshGlobalSetupFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read mesh config file %v: %v", meshGlobalSetupFile, err)
		}
		fileMeshContents = string(contents)
	}
	meshGlobalSetup, err := getMeshGlobalSetup(fileMeshContents, annotations["proxy.dubbo.apache.org/config"], proxyConfigEnv)
	if err != nil {
		return nil, err
	}
	proxyConfig := mesh.DefaultProxyConfig()
	if meshGlobalSetup.DefaultConfig != nil {
		proxyConfig = meshGlobalSetup.DefaultConfig
	}
	// TODO ResolveAddr
	// TODO ValidateMeshGlobalSetupProxyConfig
	return proxyConfig, nil
}

func getMeshGlobalSetup(fileOverride, annotationOverride, proxyConfigEnv string) (*meshv1alpha1.MeshGlobalSetup, error) {
	mc := mesh.DefaultMeshGlobalSetup()
	if fileOverride != "" {
		log.Infof("Apply mesh global setup from file %v", fileOverride)
		fileMesh, err := mesh.ApplyMeshGlobalSetup(fileOverride, mc)
		if err != nil || fileMesh == nil {
			return nil, fmt.Errorf("failed to unmarshal mesh config from file [%v]: %v", fileOverride, err)
		}
		mc = fileMesh
	}

	// Original order: env first, then annotation
	if proxyConfigEnv != "" {
		log.Infof("Apply proxy config from env %v", proxyConfigEnv)
		envMesh, err := mesh.ApplyProxyConfig(proxyConfigEnv, mc)
		if err != nil || envMesh == nil {
			return nil, fmt.Errorf("failed to unmarshal mesh config from environment [%v]: %v", proxyConfigEnv, err)
		}
		mc = envMesh
	}

	if annotationOverride != "" {
		log.Infof("Apply proxy config from annotation %v", annotationOverride)
		annotationMesh, err := mesh.ApplyProxyConfig(annotationOverride, mc)
		if err != nil || annotationMesh == nil {
			return nil, fmt.Errorf("failed to unmarshal mesh config from annotation [%v]: %v", annotationOverride, err)
		}
		mc = annotationMesh
	}

	return mc, nil
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}
