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

package grpcgen

import (
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	v3 "github.com/apache/dubbo-kubernetes/sail/pkg/xds/v3"
)

type GrpcConfigGenerator struct{}

func (g *GrpcConfigGenerator) Generate(proxy *model.Proxy, w *model.WatchedResource, req *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	// Extract requested resource names from WatchedResource
	// If ResourceNames is empty (wildcard request), pass empty slice
	// BuildListeners will handle empty names as wildcard request and generate all listeners
	var requestedNames []string
	if w != nil && w.ResourceNames != nil && len(w.ResourceNames) > 0 {
		requestedNames = w.ResourceNames.UnsortedList()
	}

	switch w.TypeUrl {
	case v3.ListenerType:
		// Pass requested names to BuildListeners to ensure consistent behavior
		// This is critical for gRPC proxyless clients to avoid resource count oscillation
		// When requestedNames is empty (wildcard), BuildListeners generates all listeners
		// When requestedNames is non-empty, BuildListeners only generates requested listeners
		return g.BuildListeners(proxy, req.Push, requestedNames), model.DefaultXdsLogDetails, nil
	case v3.ClusterType:
		return g.BuildClusters(proxy, req.Push, requestedNames), model.DefaultXdsLogDetails, nil
	case v3.RouteType:
		return g.BuildHTTPRoutes(proxy, req.Push, requestedNames), model.DefaultXdsLogDetails, nil
	}

	return nil, model.DefaultXdsLogDetails, nil
}
