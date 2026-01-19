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

package xds

import (
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/model"
)

type StatusGen struct {
	Server *DiscoveryServer
}

func NewStatusGen(s *DiscoveryServer) *StatusGen {
	return &StatusGen{
		Server: s,
	}
}

func (sg *StatusGen) Generate(proxy *model.Proxy, w *model.WatchedResource, req *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	return sg.handleInternalRequest(proxy, w, req)
}

func (sg *StatusGen) GenerateDeltas(proxy *model.Proxy, req *model.PushRequest, w *model.WatchedResource) (model.Resources, model.DeletedResources, model.XdsLogDetails, bool, error) {
	res, detail, err := sg.handleInternalRequest(proxy, w, req)
	return res, nil, detail, true, err
}

func (sg *StatusGen) handleInternalRequest(_ *model.Proxy, _ *model.WatchedResource, _ *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	res := model.Resources{}
	return res, model.DefaultXdsLogDetails, nil
}
