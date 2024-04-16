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

package model

import (
	"github.com/apache/dubbo-kubernetes/pkg/core/managers/apis/dataplane"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
)

type SearchInstanceReq struct {
	AppName string `form:"appName"`
	PageReq
}

type SearchInstanceResp struct {
	CPU              string            `json:"cpu"`
	DeployCluster    string            `json:"deployCluster"`
	DeployState      State             `json:"deployState"`
	IP               string            `json:"ip"`
	Labels           map[string]string `json:"labels"`
	Memory           string            `json:"memory"`
	Name             string            `json:"name"`
	RegisterClusters []string          `json:"registerClusters"`
	RegisterStates   []State           `json:"registerStates"`
	RegisterTime     string            `json:"registerTime"`
	StartTime        string            `json:"startTime"`
}

func (r *SearchInstanceResp) FromDataplaneResource(dr *mesh.DataplaneResource) *SearchInstanceResp {
	// TODO: support more fields
	r.IP = dr.GetIP()

	meta := dr.GetMeta()
	r.Name = meta.GetName()
	r.StartTime = meta.GetCreationTime().String()
	r.Labels = meta.GetLabels() // FIXME: in k8s mode, additional labels are append in KubernetesMetaAdapter.GetLabels

	spec := dr.Spec
	{
		statusValue := spec.Extensions[dataplane.ExtensionsPodPhaseKey]
		if v, ok := spec.Extensions[dataplane.ExtensionsPodStatusKey]; ok {
			statusValue = v
		}
		if v, ok := spec.Extensions[dataplane.ExtensionsContainerStatusReasonKey]; ok {
			statusValue = v
		}
		r.DeployState = State{
			Value: statusValue,
		}
	}

	return r
}

type State struct {
	Label string `json:"label"`
	Level string `json:"level"`
	Tip   string `json:"tip"`
	Value string `json:"value"`
}
