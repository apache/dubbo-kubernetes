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
	"strconv"

	"github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/managers/apis/dataplane"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
)

type ApplicationDetailReq struct {
	AppName string `form:"appName"`
}

type ApplicationDetailResp struct {
	AppName          string   `json:"appName"`
	AppTypes         []string `json:"appTypes"`
	DeployClusters   []string `json:"deployClusters"`
	DubboPorts       []string `json:"dubboPorts"`
	DubboVersions    []string `json:"dubboVersions"`
	Images           []string `json:"images"`
	RegisterClusters []string `json:"registerClusters"`
	RegisterModes    []string `json:"registerModes"`
	RPCProtocols     []string `json:"rpcProtocols"`
	SerialProtocols  []string `json:"serialProtocols"`
	Workloads        []string `json:"workloads"`
}

func (r *ApplicationDetailResp) FromApplicationDetail(ad *ApplicationDetail) *ApplicationDetailResp {
	r.AppTypes = ad.AppTypes.Values()
	r.DeployClusters = ad.DeployClusters.Values()
	r.DubboPorts = ad.DubboPorts.Values()
	r.DubboVersions = ad.DubboVersions.Values()
	r.Images = ad.Images.Values()
	r.RegisterClusters = ad.RegisterClusters.Values()
	r.RegisterModes = ad.RegisterModes.Values()
	r.RPCProtocols = ad.RPCProtocols.Values()
	r.SerialProtocols = ad.SerialProtocols.Values()
	r.Workloads = ad.Workloads.Values()
	return r
}

type ApplicationDetail struct {
	AppTypes         Set
	DeployClusters   Set
	DubboPorts       Set
	DubboVersions    Set
	Images           Set
	RegisterClusters Set
	RegisterModes    Set
	RPCProtocols     Set
	SerialProtocols  Set
	Workloads        Set
}

func NewApplicationDetail() *ApplicationDetail {
	return &ApplicationDetail{
		AppTypes:         NewSet(),
		DeployClusters:   NewSet(),
		DubboPorts:       NewSet(),
		DubboVersions:    NewSet(),
		Images:           NewSet(),
		RegisterClusters: NewSet(),
		RegisterModes:    NewSet(),
		RPCProtocols:     NewSet(),
		SerialProtocols:  NewSet(),
		Workloads:        NewSet(),
	}
}

func (a *ApplicationDetail) Merge(dataplane *mesh.DataplaneResource) {
	// TODO: support more fields
	inbounds := dataplane.Spec.Networking.Inbound
	for _, inbound := range inbounds {
		a.mergeInbound(inbound)
	}
	extensions := dataplane.Spec.Extensions
	a.mergeExtensions(extensions)
}

func (a *ApplicationDetail) mergeInbound(inbound *v1alpha1.Dataplane_Networking_Inbound) {
	a.DubboPorts.Add(strconv.Itoa(int(inbound.Port)))
	a.RPCProtocols.Add(inbound.Tags[v1alpha1.ProtocolTag])
	a.DeployClusters.Add(inbound.Tags[v1alpha1.ZoneTag])
}

func (a *ApplicationDetail) mergeExtensions(extensions map[string]string) {
	image := extensions[dataplane.ExtensionsImageKey]
	a.Images.Add(image)
}

type ApplicationTabInstanceInfoReq struct {
	AppName string `form:"appName"`
}

type ApplicationTabInstanceInfoResp struct {
	AppName         string            `json:"appName"`
	CreateTime      string            `json:"createTime"`
	DeployState     string            `json:"deployState"`
	IP              string            `json:"ip"`
	Labels          map[string]string `json:"labels"`
	Name            string            `json:"name"`
	RegisterCluster string            `json:"registerCluster"`
	RegisterState   string            `json:"registerState"`
	RegisterTime    string            `json:"registerTime"`
	WorkloadName    string            `json:"workloadName"`
}

func (a *ApplicationTabInstanceInfoResp) FromDataplaneResource(dataplane *mesh.DataplaneResource) *ApplicationTabInstanceInfoResp {
	// TODO: support more fields
	a.AppName = dataplane.GetMeta().GetLabels()[v1alpha1.AppTag]
	a.CreateTime = dataplane.Meta.GetCreationTime().String()
	a.IP = dataplane.Spec.Networking.Address
	a.Labels = dataplane.GetMeta().GetLabels()
	a.Name = dataplane.Meta.GetName()
	return a
}
