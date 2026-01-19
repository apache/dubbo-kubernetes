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

package bootstrap

import (
	kubecontroller "github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/serviceregistry/kube/controller"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/ctrlz"
	"github.com/apache/dubbo-kubernetes/pkg/env"
	"github.com/apache/dubbo-kubernetes/pkg/keepalive"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
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
}

type InjectionOptions struct {
	// Directory of injection related config files.
	InjectionDirectory string
}

type PlanetArgs struct {
	ServerOptions        DiscoveryServerOptions
	RegistryOptions      RegistryOptions
	InjectionOptions     InjectionOptions
	MeshGlobalConfigFile string
	PodName              string
	Namespace            string
	CtrlZOptions         *ctrlz.Options
	KeepaliveOptions     *keepalive.Options
	KrtDebugger          *krt.DebugHandler `json:"-"`
	Revision             string
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

func NewPlanetArgs(initFuncs ...func(*PlanetArgs)) *PlanetArgs {
	p := &PlanetArgs{}

	// Apply Default Values.
	p.applyDefaults()

	// Apply custom initialization functions.
	for _, fn := range initFuncs {
		fn(p)
	}

	return p
}

var Revision = env.Register("REVISION", "", "").Get()

func (p *PlanetArgs) applyDefaults() {
	p.Namespace = PodNamespace
	p.PodName = PodName
	p.Revision = Revision
	p.KeepaliveOptions = keepalive.DefaultOption()
	p.RegistryOptions.ClusterRegistriesNamespace = p.Namespace
}
