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
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
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
	Ip              string            `json:"ip"`
	Name            string            `json:"name"`
	WorkloadName    string            `json:"workloadName"`
	AppName         string            `json:"appName"`
	DeployState     string            `json:"deployState"`
	DeployCluster   string            `json:"deployCluster"`
	RegisterState   string            `json:"registerState"`
	RegisterCluster string            `json:"registerCluster"`
	CreateTime      string            `json:"createTime"`
	RegisterTime    string            `json:"registerTime"` // TODO: not converted
	Labels          map[string]string `json:"labels"`
}

func (r *SearchInstanceResp) FromDataplaneResource(dr *mesh.DataplaneResource) *SearchInstanceResp {
	// TODO: support more fields
	r.Ip = dr.GetIP()
	meta := dr.GetMeta()
	r.Name = meta.GetName()
	r.CreateTime = meta.GetCreationTime().String()
	r.RegisterTime = r.CreateTime // TODO: seperate createTime and RegisterTime
	r.RegisterCluster = dr.Spec.Networking.Inbound[0].Tags[v1alpha1.ZoneTag]
	r.DeployCluster = r.RegisterCluster
	if r.RegisterTime != "" {
		r.RegisterState = "Registed"
	} else {
		r.RegisterState = "UnRegisted"
	}
	// label conversion
	r.Labels = meta.GetLabels()
	// spec conversion
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
		// name field source is different between universal and k8s mode
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
	Labels           map[string]string `json:"labels"`
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

type ProbeStruct struct {
	StartupProbe   StartupProbe   `json:"startupProbe"`
	ReadinessProbe ReadinessProbe `json:"readinessProbe"`
	LivenessProbe  LivenessProbe  `json:"livenessProbe"`
}

type StartupProbe struct {
	Type string `json:"type"`
	Port int    `json:"port"`
	Open bool   `json:"open"`
}
type ReadinessProbe struct {
	Type string `json:"type"`
	Port int    `json:"port"`
	Open bool   `json:"open"`
}
type LivenessProbe struct {
	Type string `json:"type"`
	Port int    `json:"port"`
	Open bool   `json:"open"`
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
	r.Probes = id.Probes
	return r
}

type InstanceDetail struct {
	RpcPort          int
	Ip               string
	AppName          string
	WorkloadName     string
	Labels           map[string]string
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
	Probes           ProbeStruct
}

func NewInstanceDetail() *InstanceDetail {
	return &InstanceDetail{
		RpcPort:          -1,
		Ip:               "",
		AppName:          "",
		WorkloadName:     "",
		Labels:           nil,
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
	probes := dataplane.Spec.Probes
	a.mergeProbes(probes)

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
		a.DeployCluster = deploycluster // TODO: seperate deployCluster and registerCluster
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
	a.Node = extensions[dataplane.ExtensionsNodeNameKey]
}

func (a *InstanceDetail) mergeMeta(meta model.ResourceMeta) {
	a.CreateTime = meta.GetCreationTime().String()
	a.RegisterTime = meta.GetModificationTime().String() // Not sure if it's the right field
	a.ReadyTime = a.RegisterTime
	// TODO: seperate createTime , RegisterTime and ReadyTime
	a.Labels = meta.GetLabels()
}

func (a *InstanceDetail) mergeProbes(probes *mesh_proto.Dataplane_Probes) {
	if probes == nil {
		return
	}
	portStartup := probes.Endpoints[0].InboundPort
	portReadiness := probes.Endpoints[1].InboundPort
	portLiveness := probes.Endpoints[2].InboundPort
	a.Probes = ProbeStruct{
		StartupProbe: StartupProbe{
			Type: "HTTP", // TODO: support more scheme

			Port: int(portStartup),
			Open: true,
		},
		ReadinessProbe: ReadinessProbe{
			Type: "HTTP", // TODO: support more scheme
			Port: int(portReadiness),
			Open: true,
		},
		LivenessProbe: LivenessProbe{
			Type: "HTTP", // TODO: support more scheme
			Port: int(portLiveness),
			Open: true,
		},
	}
}
