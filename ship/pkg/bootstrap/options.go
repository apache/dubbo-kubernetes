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
	kubecontroller "github.com/apache/dubbo-kubernetes/ship/pkg/serviceregistry/kube/controller"
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

type ShipArgs struct {
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
}

func NewShipArgs(initFuncs ...func(*ShipArgs)) *ShipArgs {
	p := &ShipArgs{}

	// Apply Default Values.
	p.applyDefaults()

	// Apply custom initialization functions.
	for _, fn := range initFuncs {
		fn(p)
	}

	return p
}

func (p *ShipArgs) applyDefaults() {
	p.Namespace = PodNamespace
	p.PodName = PodName
	p.KeepaliveOptions = keepalive.DefaultOption()
	p.RegistryOptions.ClusterRegistriesNamespace = p.Namespace
}
