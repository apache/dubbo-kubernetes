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

package bootstrap

import (
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/ctrlz"
	"github.com/apache/dubbo-kubernetes/pkg/env"
	"github.com/apache/dubbo-kubernetes/pkg/keepalive"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	kubecontroller "github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/kube/controller"
)

var (
	PodNamespace = env.Register("POD_NAMESPACE", constants.DubboSystemNamespace, "").Get()
	PodName      = env.Register("POD_NAME", "", "").Get()
)

type RegistryOptions struct {
	FileDir                    string
	Registries                 []string
	KubeConfig                 string
	KubeOptions                kubecontroller.Options
	ClusterRegistriesNamespace string
	// Multi-cluster configuration
	MultiCluster MultiClusterOptions `json:"multiCluster" yaml:"multiCluster"`
	// Multi-registry configuration
	MultiRegistry MultiRegistryOptions `json:"multiRegistry" yaml:"multiRegistry"`
}

// MultiClusterOptions holds configuration for managing multiple Kubernetes clusters
type MultiClusterOptions struct {
	// Enabled indicates if multi-cluster support is enabled
	Enabled bool `json:"enabled" yaml:"enabled"`
	// Clusters is a list of cluster configurations
	Clusters []ClusterConfig `json:"clusters" yaml:"clusters"`
}

// ClusterConfig holds configuration for a single Kubernetes cluster
type ClusterConfig struct {
	// ID is the unique identifier for this cluster
	ID string `json:"id" yaml:"id"`
	// Name is a human-readable name for this cluster
	Name string `json:"name" yaml:"name"`
	// KubeConfig is the path to kubeconfig file for this cluster
	KubeConfig string `json:"kubeConfig" yaml:"kubeConfig"`
	// Context is the context name to use from kubeconfig
	Context string `json:"context" yaml:"context"`
	// Enabled indicates if this cluster is enabled
	Enabled bool `json:"enabled" yaml:"enabled"`
	// Priority determines the priority of this cluster (higher value = higher priority)
	Priority int `json:"priority" yaml:"priority"`
	// Region specifies the region this cluster is located in
	Region string `json:"region" yaml:"region"`
	// Zone specifies the zone this cluster is located in
	Zone string `json:"zone" yaml:"zone"`
}

// MultiRegistryOptions holds configuration for managing multiple service registries
type MultiRegistryOptions struct {
	// Enabled indicates if multi-registry support is enabled
	Enabled bool `json:"enabled" yaml:"enabled"`
	// Registries is a list of registry configurations
	Registries []RegistryConfig `json:"registries" yaml:"registries"`
}

// RegistryConfig holds configuration for a single service registry
type RegistryConfig struct {
	// ID is the unique identifier for this registry
	ID string `json:"id" yaml:"id"`
	// Name is a human-readable name for this registry
	Name string `json:"name" yaml:"name"`
	// Type specifies the registry type (Kubernetes, Nacos, Zookeeper)
	Type string `json:"type" yaml:"type"`
	// Address is the registry server address
	Address string `json:"address" yaml:"address"`
	// Enabled indicates if this registry is enabled
	Enabled bool `json:"enabled" yaml:"enabled"`
	// Priority determines the priority of this registry (higher value = higher priority)
	Priority int `json:"priority" yaml:"priority"`
	// Config holds registry-specific configuration
	Config map[string]interface{} `json:"config" yaml:"config"`
}

type SailArgs struct {
	ServerOptions      DiscoveryServerOptions
	RegistryOptions    RegistryOptions
	MeshConfigFile     string
	NetworksConfigFile string
	PodName            string
	Namespace          string
	CtrlZOptions       *ctrlz.Options
	KeepaliveOptions   *keepalive.Options
	KrtDebugger        *krt.DebugHandler `json:"-"`
}

type DiscoveryServerOptions struct {
	HTTPAddr       string
	HTTPSAddr      string
	GRPCAddr       string
	SecureGRPCAddr string
	TLSOptions     TLSOptions
}

type TLSOptions struct {
	// CaCertFile and related are set using CLI flags.
	CaCertFile      string
	CertFile        string
	KeyFile         string
	TLSCipherSuites []string
	CipherSuits     []uint16 // This is the parsed cipher suites
}

func NewSailArgs(initFuncs ...func(*SailArgs)) *SailArgs {
	p := &SailArgs{}

	// Apply Default Values.
	p.applyDefaults()

	// Apply custom initialization functions.
	for _, fn := range initFuncs {
		fn(p)
	}

	return p
}

func (p *SailArgs) applyDefaults() {
	p.Namespace = PodNamespace
	p.PodName = PodName
	p.KeepaliveOptions = keepalive.DefaultOption()
	p.RegistryOptions.ClusterRegistriesNamespace = p.Namespace
}
