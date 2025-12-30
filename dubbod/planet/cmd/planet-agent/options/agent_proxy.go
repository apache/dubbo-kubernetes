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

package options

import planetagent "github.com/apache/dubbo-kubernetes/pkg/dubbo-agent"

// ProxyArgs provides all of the configuration parameters for the Saku proxy.
type ProxyArgs struct {
	planetagent.Proxy
	Concurrency int
	StsPort     int

	TokenManagerPlugin string

	MeshGlobalConfigFile string
	TemplateFile         string

	PodName      string
	PodNamespace string
}

func NewProxyArgs() ProxyArgs {
	p := ProxyArgs{}
	p.applyDefaults()
	return p
}

func (node *ProxyArgs) applyDefaults() {
	node.PodName = PodNameVar.Get()
	node.PodNamespace = PodNamespaceVar.Get()
}
