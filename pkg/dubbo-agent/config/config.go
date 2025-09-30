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

package config

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/bootstrap"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/env"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"istio.io/api/annotation"
	meshconfig "istio.io/api/mesh/v1alpha1"
	"k8s.io/klog/v2"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// ConstructProxyConfig returns proxyConfig
func ConstructProxyConfig(meshConfigFile, serviceCluster, proxyConfigEnv string, concurrency int) (*meshconfig.ProxyConfig, error) {
	annotations, err := bootstrap.ReadPodAnnotations("")
	if err != nil {
		if os.IsNotExist(err) {
			klog.V(2).Infof("failed to read pod annotations: %v", err)
		} else {
			klog.V(2).Infof("failed to read pod annotations: %v", err)
		}
	}
	var fileMeshContents string
	if fileExists(meshConfigFile) {
		contents, err := os.ReadFile(meshConfigFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read mesh config file %v: %v", meshConfigFile, err)
		}
		fileMeshContents = string(contents)
	}
	meshConfig, err := getMeshConfig(fileMeshContents, annotations[annotation.ProxyConfig.Name], proxyConfigEnv)
	if err != nil {
		return nil, err
	}
	proxyConfig := mesh.DefaultProxyConfig()
	if meshConfig.DefaultConfig != nil {
		proxyConfig = meshConfig.DefaultConfig
	}

	// Concurrency wasn't explicitly set
	if proxyConfig.Concurrency == nil {
		// We want to detect based on CPU limit configured. If we are running on a 100 core machine, but with
		// only 2 CPUs allocated, we want to have 2 threads, not 100, or we will get excessively throttled.
		if CPULimit != 0 {
			klog.Infof("cpu limit detected as %v, setting concurrency", CPULimit)
			proxyConfig.Concurrency = wrapperspb.Int32(int32(CPULimit))
		}
	}
	// Respect the old flag, if they set it. This should never be set in typical installation.
	if concurrency != 0 {
		klog.V(2).Infof("legacy --concurrency=%d flag detected; prefer to use ProxyConfig", concurrency)
		proxyConfig.Concurrency = wrapperspb.Int32(int32(concurrency))
	}

	if proxyConfig.Concurrency.GetValue() == 0 {
		if CPULimit < runtime.NumCPU() {
			klog.V(2).Infof("concurrency is set to 0, which will use a thread per CPU on the host. However, CPU limit is set lower. "+
				"This is not recommended and may lead to performance issues. "+
				"CPU count: %d, CPU Limit: %d.", runtime.NumCPU(), CPULimit)
		}
	}

	if x, ok := proxyConfig.GetClusterName().(*meshconfig.ProxyConfig_ServiceCluster); ok {
		if x.ServiceCluster == "" {
			proxyConfig.ClusterName = &meshconfig.ProxyConfig_ServiceCluster{ServiceCluster: serviceCluster}
		}
	}
	// TODO ResolveAddr
	// TODO ValidateMeshConfigProxyConfig
	return applyAnnotations(proxyConfig, annotations), nil
}

func getMeshConfig(fileOverride, annotationOverride, proxyConfigEnv string) (*meshconfig.MeshConfig, error) {
	mc := mesh.DefaultMeshConfig()
	if fileOverride != "" {
		klog.Infof("Apply mesh config from file %v", fileOverride)
		fileMesh, err := mesh.ApplyMeshConfig(fileOverride, mc)
		if err != nil || fileMesh == nil {
			return nil, fmt.Errorf("failed to unmarshal mesh config from file [%v]: %v", fileOverride, err)
		}
		mc = fileMesh
	}

	if proxyConfigEnv != "" {
		klog.Infof("Apply proxy config from env %v", proxyConfigEnv)
		envMesh, err := mesh.ApplyProxyConfig(proxyConfigEnv, mc)
		if err != nil || envMesh == nil {
			return nil, fmt.Errorf("failed to unmarshal mesh config from environment [%v]: %v", proxyConfigEnv, err)
		}
		mc = envMesh
	}

	if annotationOverride != "" {
		klog.Infof("Apply proxy config from annotation %v", annotationOverride)
		annotationMesh, err := mesh.ApplyProxyConfig(annotationOverride, mc)
		if err != nil || annotationMesh == nil {
			return nil, fmt.Errorf("failed to unmarshal mesh config from annotation [%v]: %v", annotationOverride, err)
		}
		mc = annotationMesh
	}

	return mc, nil
}

// Apply any overrides to proxy config from annotations
func applyAnnotations(config *meshconfig.ProxyConfig, annos map[string]string) *meshconfig.ProxyConfig {
	if v, f := annos[annotation.SidecarDiscoveryAddress.Name]; f {
		config.DiscoveryAddress = v
	}
	if v, f := annos[annotation.SidecarStatusPort.Name]; f {
		p, err := strconv.Atoi(v)
		if err != nil {
			klog.Errorf("Invalid annotation %v=%v: %v", annotation.SidecarStatusPort.Name, v, err)
		}
		config.StatusPort = int32(p)
	}
	return config
}

func GetSailSan(discoveryAddress string) string {
	discHost := strings.Split(discoveryAddress, ":")[0]
	// For local debugging - the discoveryAddress is set to localhost, but the cert issued for normal SA.
	if discHost == "localhost" {
		discHost = "dubbod.istio-system.svc"
	}
	return discHost
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

var CPULimit = env.Register(
	"DUBBO_CPU_LIMIT",
	0,
	"CPU limit for the current process. Expressed as an integer value, rounded up.",
).Get()
