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
	"github.com/apache/dubbo-kubernetes/pkg/core/managers/apis/dataplane"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"strconv"
)

type SearchInstanceReq struct {
	AppName string `form:"appName"`
	PageReq
}

type InstanceDetailReq struct {
	InstanceName string `form:"instanceName"`
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

type InstanceDetailResp struct {
	RpcPort          string            `json:"rpcPort"`
	Ip               string            `json:"ip"`
	AppName          string            `json:"appName"`
	WorkloadName     string            `json:"workloadName"`
	Labels           []string          `json:"labels"`
	CreateTime       string            `json:"createTime"`
	ReadyTime        string            `json:"readyTime"`
	RegisterTime     string            `json:"registerTime"`
	RegisterClusters []string          `json:"registerClusters"`
	DeployCluster    string            `json:"deployCluster"`
	Node             string            `json:"node"`
	Image            string            `json:"image"`
	Probes           ProbeStruct       `json:"probes"`
	Tags             map[string]string `json:"tags"`
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
	r.Labels = id.Labels.Values()
	r.CreateTime = id.CreateTime
	r.ReadyTime = id.ReadyTime
	r.RegisterTime = id.RegisterTime
	r.RegisterClusters = id.RegisterClusters.Values()
	r.DeployCluster = id.DeployCluster
	r.Node = id.Node
	r.Image = id.Image
	r.Tags = id.Tags
	return r
}

type InstanceDetail struct {
	RpcPort          string
	Ip               string
	AppName          string
	WorkloadName     string
	Labels           Set
	CreateTime       string
	ReadyTime        string
	RegisterTime     string
	RegisterClusters Set
	DeployCluster    string
	Node             string
	Image            string
	Tags             map[string]string
}

func NewInstanceDetail() *InstanceDetail {
	return &InstanceDetail{
		RpcPort:          "",
		Ip:               "",
		AppName:          "",
		WorkloadName:     "",
		Labels:           NewSet(),
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

}

func (a *InstanceDetail) mergeInbound(inbound *v1alpha1.Dataplane_Networking_Inbound) {
	a.RpcPort = strconv.Itoa(int(inbound.Port))
	a.RegisterClusters.Add(inbound.Tags[v1alpha1.ZoneTag])
	a.Tags = inbound.Tags
}

func (a *InstanceDetail) mergeExtensions(extensions map[string]string) {
	image := extensions[dataplane.ExtensionsImageKey]
	a.Image = image
	a.AppName = extensions[dataplane.ExtensionApplicationNameKey]
}

func (a *InstanceDetail) mergeMeta(meta model.ResourceMeta) {
	a.CreateTime = meta.GetCreationTime().String()
	a.RegisterTime = meta.GetModificationTime().String() //Not sure if it's the right field

}
