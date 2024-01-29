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

package sync

import (
	"context"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	xds_context "github.com/apache/dubbo-kubernetes/pkg/xds/context"
)

type IngressProxyBuilder struct {
	ResManager manager.ResourceManager

	apiVersion        core_xds.APIVersion
	zone              string
	ingressTagFilters []string
}

func (p *IngressProxyBuilder) Build(
	ctx context.Context,
	key core_model.ResourceKey,
	aggregatedMeshCtxs xds_context.MeshContext,
) (*core_xds.Proxy, error) {
	return &core_xds.Proxy{}, nil
}
