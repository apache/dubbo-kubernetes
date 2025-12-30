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

package core

import (
	serviceRouteIndex "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/model"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
)

type ConfigGenerator interface {
	BuildListeners(node *model.Proxy, push *model.PushContext) []*listener.Listener
	BuildClusters(node *model.Proxy, req *model.PushRequest) ([]*discovery.Resource, model.XdsLogDetails)
	BuildDeltaClusters(proxy *model.Proxy, updates *model.PushRequest,
		watched *model.WatchedResource) ([]*discovery.Resource, []string, model.XdsLogDetails, bool)
	BuildHTTPRoutes(node *model.Proxy, req *model.PushRequest, routeNames []string) ([]*discovery.Resource, model.XdsLogDetails)
	BuildExtensionConfiguration(node *model.Proxy, push *model.PushContext, extensionConfigNames []string,
		pullSecrets map[string][]byte) []*core.TypedExtensionConfig
	serviceRouteIndexChanged(mesh *serviceRouteIndex.MeshGlobalConfig)
}

type ConfigGeneratorImpl struct {
	Cache model.XdsCache
}

func NewConfigGenerator(cache model.XdsCache) *ConfigGeneratorImpl {
	return &ConfigGeneratorImpl{
		Cache: cache,
	}
}

func (configgen *ConfigGeneratorImpl) serviceRouteIndexChanged(_ *serviceRouteIndex.MeshGlobalConfig) {
	return
}
