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
	kubecontroller "github.com/apache/dubbo-kubernetes/navigator/pkg/serviceregistry/kube/controller"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/env"
)

type RegistryOptions struct {
	FileDir                    string
	Registries                 []string
	KubeConfig                 string
	KubeOptions                kubecontroller.Options
	ClusterRegistriesNamespace string
}

type NaviArgs struct {
	ServerOptions   DiscoveryServerOptions
	RegistryOptions RegistryOptions
	MeshConfigFile  string
	PodName         string
	Namespace       string
}

type DiscoveryServerOptions struct {
	HTTPAddr       string
	HTTPSAddr      string
	GRPCAddr       string
	SecureGRPCAddr string
}

var (
	PodNamespace = env.Register("POD_NAMESPACE", constants.DubboSystemNamespace, "").Get()
	PodName      = env.Register("POD_NAME", "", "").Get()
)

func (p *NaviArgs) applyDefaults() {
	p.Namespace = PodNamespace
	p.PodName = PodName
	p.RegistryOptions.ClusterRegistriesNamespace = p.Namespace
}
