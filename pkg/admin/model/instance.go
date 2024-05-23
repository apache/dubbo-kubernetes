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
	"github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/admin/constants"
	"github.com/apache/dubbo-kubernetes/pkg/core/managers/apis/dataplane"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
)

type SearchInstanceReq struct {
	AppName string `form:"appName"`
	PageReq
}

type InstanceDetailReq struct {
	InstanceName string `form:"instanceName"`
}

type SearchInstanceResp struct {
	Ip              string `json:"ip"`
	Name            string `json:"name"`
	WorkloadName    string `json:"workloadName"`
	AppName         string `json:"appName"`
	DeployState     string `json:"deployState"`
	DeployCluster   string `json:"deployCluster"`
	RegisterState   string `json:"registerState"`
	RegisterCluster string `json:"registerCluster"`
	CreateTime      string `json:"createTime"`
	RegisterTime    string `json:"registerTime"` //TODO: not converted
	Labels          struct {
		Region  string `json:"region"`
		Version string `json:"version"`
	} `json:"labels"`
}

func (r *SearchInstanceResp) FromDataplaneResource(dr *mesh.DataplaneResource) *SearchInstanceResp {
	// TODO: support more fields
	r.Ip = dr.GetIP()
	meta := dr.GetMeta()
	r.Name = meta.GetName()
	r.CreateTime = meta.GetCreationTime().String()
	r.RegisterTime = r.CreateTime //TODO: seperate createTime and RegisterTime
	r.RegisterCluster = dr.Spec.Networking.Inbound[0].Tags[v1alpha1.ZoneTag]
	r.DeployCluster = r.RegisterCluster
	if r.RegisterTime != "" {
		r.RegisterState = "Registed"
	} else {
		r.RegisterState = "UnRegisted"
	}
	//label conversion
	labels := meta.GetLabels() // FIXME: in k8s mode, additional labels are append in KubernetesMetaAdapter.GetLabels
	r.Labels.Region = labels[constants.RegionKey]
	r.Labels.Version = labels[constants.DubboVersionKey]
	//spec conversion
	spec := dr.Spec
	{
		statusValue := spec.Extensions[dataplane.ExtensionsPodPhaseKey]
		if v, ok := spec.Extensions[dataplane.ExtensionsPodStatusKey]; ok {
			statusValue = v
		}
		if v, ok := spec.Extensions[dataplane.ExtensionsContainerStatusReasonKey]; ok {
			statusValue = v
		}
		r.DeployState = statusValue
		r.WorkloadName = spec.Extensions[dataplane.ExtensionsWorkLoadKey]
		//name field source is different between universal and k8s mode
		r.AppName = spec.Extensions[dataplane.ExtensionApplicationNameKey]
		if r.AppName == "" {
			for _, inbound := range spec.Networking.Inbound {
				r.AppName = inbound.Tags[v1alpha1.AppTag]
			}
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

type InstanceDetailResp struct {
	RpcPort          int               `json:"rpcPort"`
	Ip               string            `json:"ip"`
	AppName          string            `json:"appName"`
	WorkloadName     string            `json:"workloadName"`
	Labels           LabelStruct       `json:"labels"`
	CreateTime       string            `json:"createTime"`
	ReadyTime        string            `json:"readyTime"`
	RegisterTime     string            `json:"registerTime"`
	RegisterClusters []string          `json:"registerClusters"`
	DeployCluster    string            `json:"deployCluster"`
	DeployState      string            `json:"deployState"`
	RegisterStates   string            `json:"registerStates"`
	Node             string            `json:"node"`
	Image            string            `json:"image"`
	Probes           ProbeStruct       `json:"probes"`
	Tags             map[string]string `json:"tags"`
}

type LabelStruct struct {
	App     string `json:"app"`
	Version string `json:"version"`
	Region  string `json:"region"`
}

type ProbeStruct struct {
	StartupProbe struct {
		Type string `json:"type"`
		Port int    `json:"port"`
		Open bool   `json:"open"`
	} `json:"startupProbe"`
	ReadinessProbe struct {
		Type string `json:"type"`
		Port int    `json:"port"`
		Open bool   `json:"open"`
	} `json:"readinessProbe"`
	LivenessProbe struct {
		Type string `json:"type"`
		Port int    `json:"port"`
		Open bool   `json:"open"`
	} `json:"livenessProbe"`
}

func (r *InstanceDetailResp) FromInstanceDetail(id *InstanceDetail) *InstanceDetailResp {
	r.AppName = id.AppName
	r.RpcPort = id.RpcPort
	r.Ip = id.Ip
	r.WorkloadName = id.WorkloadName
	r.Labels = id.Labels
	r.CreateTime = id.CreateTime
	r.ReadyTime = id.ReadyTime
	r.RegisterTime = id.RegisterTime
	r.RegisterClusters = id.RegisterClusters.Values()
	r.DeployCluster = id.DeployCluster
	r.DeployCluster = id.DeployCluster
	r.DeployState = id.DeployState
	r.Node = id.Node
	r.Image = id.Image
	r.Tags = id.Tags
	r.RegisterStates = id.RegisterState
	return r
}

type InstanceDetail struct {
	RpcPort          int
	Ip               string
	AppName          string
	WorkloadName     string
	Labels           LabelStruct
	CreateTime       string
	ReadyTime        string
	RegisterTime     string
	RegisterState    string
	RegisterClusters Set
	DeployCluster    string
	DeployState      string
	Node             string
	Image            string
	Tags             map[string]string
}

func NewInstanceDetail() *InstanceDetail {
	return &InstanceDetail{
		RpcPort:          -1,
		Ip:               "",
		AppName:          "",
		WorkloadName:     "",
		Labels:           LabelStruct{},
		CreateTime:       "",
		ReadyTime:        "",
		RegisterTime:     "",
		RegisterClusters: NewSet(),
		DeployCluster:    "",
		Node:             "",
		Image:            "",
	}
}

func (a *InstanceDetail) Merge(dataplane *mesh.DataplaneResource) {
	// TODO: support more fields
	inbounds := dataplane.Spec.Networking.Inbound
	for _, inbound := range inbounds {
		a.mergeInbound(inbound)
	}
	meta := dataplane.Meta
	a.mergeMeta(meta)
	extensions := dataplane.Spec.Extensions
	a.mergeExtensions(extensions)
	a.Ip = dataplane.GetIP()
	if a.RegisterTime != "" {
		a.RegisterState = "Registed"
	} else {
		a.RegisterState = "UnRegisted"
	}
}

func (a *InstanceDetail) mergeInbound(inbound *v1alpha1.Dataplane_Networking_Inbound) {
	a.RpcPort = int(inbound.Port)
	a.RegisterClusters.Add(inbound.Tags[v1alpha1.ZoneTag])
	for _, deploycluster := range a.RegisterClusters.Values() {
		a.DeployCluster = deploycluster //TODO: seperate deployCluster and registerCluster
	}
	a.Tags = inbound.Tags
	if a.AppName == "" {
		a.AppName = inbound.Tags[v1alpha1.AppTag]
	}
}

func (a *InstanceDetail) mergeExtensions(extensions map[string]string) {
	image := extensions[dataplane.ExtensionsImageKey]
	a.Image = image
	if a.AppName == "" {
		a.AppName = extensions[dataplane.ExtensionApplicationNameKey]
	}
	a.WorkloadName = extensions[dataplane.ExtensionsWorkLoadKey]
	a.DeployState = extensions[dataplane.ExtensionsPodPhaseKey]

}

func (a *InstanceDetail) mergeMeta(meta model.ResourceMeta) {
	a.CreateTime = meta.GetCreationTime().String()
	a.RegisterTime = meta.GetModificationTime().String() //Not sure if it's the right field
	a.ReadyTime = a.RegisterState
	//TODO: seperate createTime , RegisterTime and ReadyTime
}
