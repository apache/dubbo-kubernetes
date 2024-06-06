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
	"fmt"
	"strconv"
	"strings"
)

import (
	"github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/admin/constants"
	"github.com/apache/dubbo-kubernetes/pkg/config/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/managers/apis/dataplane"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

// Todo Application Detail
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

func (a *ApplicationDetail) MergeMetaData(metadata *mesh.MetaDataResource) {
	a.mergeServiceInfo(metadata)
}

func (a *ApplicationDetail) mergeServiceInfo(metadata *mesh.MetaDataResource) {
	for _, serviceInfo := range metadata.Spec.Services {
		a.DubboVersions.Add(fmt.Sprintf("dubbo %s", serviceInfo.Params[constants.DubboVersionKey]))
		a.SerialProtocols.Add(serviceInfo.Params[constants.SerializationKey])
	}
}

func (a *ApplicationDetail) MergeDatapalne(dataplane *mesh.DataplaneResource) {
	// TODO: support more fields
	a.AppTypes.Add("无状态")
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
	a.Images.Add(extensions[dataplane.ExtensionsImageKey])
	a.Workloads.Add(extensions[dataplane.ExtensionsWorkLoadKey])
}

func (a *ApplicationDetail) GetRegistry(rt core_runtime.Runtime) {
	runtimeMode := rt.GetDeployMode()
	if runtimeMode == core.KubernetesMode {
		// In Kubernetes mode, registry cluster is the Kubernetes cluster itself
		a.RegisterClusters.Add(a.DeployClusters.Values()...)
		a.RegisterModes.Add("应用级", "接口级")
	} else if runtimeMode == core.HalfHostMode || runtimeMode == core.UniversalMode {
		// In half or universal mode, registry cluster is the zookeeper cluster or other registry center
		registryURL := rt.RegistryCenter().GetURL()
		registryCluster := registryURL.GetParam(constants.RegistryClusterKey, "")
		a.RegisterClusters.Add(registryCluster)

		registryMode := registryURL.GetParam(constants.RegisterModeKey, constants.DefaultRegisterModeAll)
		if registryMode == constants.DefaultRegisterModeAll {
			a.RegisterModes.Add("应用级", "接口级")
		} else if registryCluster == constants.DefaultRegisterModeInstance {
			a.RegisterClusters.Add("应用级")
		} else if registryCluster == constants.DefaultRegisterModeInterface {
			a.RegisterClusters.Add("接口级")
		}
	}
}

// todo Application instance info

type ApplicationTabInstanceInfoReq struct {
	AppName string `form:"appName"`
}

type ApplicationTabInstanceInfoResp struct {
	AppName         string            `json:"appName"`
	CreateTime      string            `json:"createTime"`
	DeployState     string            `json:"deployState"`
	DeployClusters  string            `json:"deployClusters"`
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
	extensions := dataplane.Spec.Extensions
	a.mergeExtension(extensions)
	a.mergeMainDataplane(dataplane)
	return a
}

// nolint
func (a *ApplicationTabInstanceInfoResp) mergeMainDataplane(dataplane *mesh.DataplaneResource) {
	a.AppName = dataplane.GetMeta().GetLabels()[v1alpha1.AppTag]
	a.CreateTime = dataplane.Meta.GetCreationTime().String()
	a.IP = dataplane.Spec.Networking.Address
	a.DeployClusters = dataplane.Spec.Networking.Inbound[0].Tags[v1alpha1.ZoneTag]
	a.Labels = dataplane.GetMeta().GetLabels()
	a.Name = dataplane.Meta.GetName()
	a.RegisterTime = a.CreateTime
	if a.RegisterTime != "" {
		a.RegisterState = "Registed"
	} else {
		a.RegisterState = "UnRegisted"
	}
}

func (a *ApplicationTabInstanceInfoResp) mergeExtension(extensions map[string]string) {
	a.WorkloadName = extensions[dataplane.ExtensionsWorkLoadKey]
	a.DeployState = extensions[dataplane.ExtensionsPodPhaseKey]
}

func (a *ApplicationTabInstanceInfoResp) GetRegistry(rt core_runtime.Runtime) {
	runtimeMode := rt.GetDeployMode()
	if runtimeMode == core.KubernetesMode {
		// In Kubernetes mode, registry cluster is the Kubernetes cluster itself
		a.RegisterCluster = a.DeployClusters
	} else if runtimeMode == core.HalfHostMode || runtimeMode == core.UniversalMode {
		// In half or universal mode, registry cluster is the zookeeper cluster or other registry center
		registryURL := rt.RegistryCenter().GetURL()
		registryCluster := registryURL.GetParam(constants.RegistryClusterKey, "")
		a.RegisterCluster = registryCluster
	}
}

// Todo Application Service

type ApplicationServiceReq struct {
	AppName string `json:"appName"`
}

type ApplicationServiceResp struct {
	ConsumeNum int64 `json:"consumeNum"`
	ProvideNum int64 `json:"provideNum"`
}

type ApplicationServiceFormReq struct {
	AppName string `json:"appName"`
	Side    string `json:"side"`
}

type ApplicationServiceFormResp struct {
	ServiceName   string         `json:"serviceName"`
	VersionGroups []versionGroup `json:"versionGroups"`
}

type versionGroup struct {
	Group   string `json:"group"`
	Version string `json:"version"`
}

func NewApplicationServiceFormResp() *ApplicationServiceFormResp {
	return &ApplicationServiceFormResp{
		ServiceName:   "",
		VersionGroups: nil,
	}
}

func (a *ApplicationServiceFormResp) FromApplicationServiceForm(form *ApplicationServiceForm) {
	a.ServiceName = form.ServiceName
	versionGroupList := make([]versionGroup, 0)
	for _, gv := range form.VersionGroups.Values() {
		groupAndVersion := strings.Split(gv, " ")
		versionGroupList = append(versionGroupList, versionGroup{
			Group:   groupAndVersion[0],
			Version: groupAndVersion[1],
		})
	}
	a.VersionGroups = versionGroupList
}

type ApplicationServiceForm struct {
	ServiceName   string
	VersionGroups Set
}

func NewApplicationServiceForm(serviceName string) *ApplicationServiceForm {
	return &ApplicationServiceForm{
		ServiceName:   serviceName,
		VersionGroups: NewSet(),
	}
}

func (a *ApplicationServiceForm) FromServiceInfo(serviceInfo *v1alpha1.ServiceInfo) {
	a.VersionGroups.Add(serviceInfo.Group + " " + serviceInfo.Version)
}

type ApplicationSearchResp struct {
	AppName          string   `json:"appName"`
	DeployClusters   []string `json:"deployClusters"`
	InstanceCount    int64    `json:"instanceCount"`
	RegistryClusters []string `json:"registryClusters"`
}

func (a *ApplicationSearchResp) FromApplicationSearch(applicationSearch *ApplicationSearch) *ApplicationSearchResp {
	a.RegistryClusters = applicationSearch.RegistryClusters.Values()
	a.InstanceCount = applicationSearch.InstanceCount
	a.DeployClusters = applicationSearch.DeployClusters.Values()
	return a
}

// Todo Application Search

type ApplicationSearch struct {
	AppName          string
	DeployClusters   Set
	InstanceCount    int64
	RegistryClusters Set
}

func NewApplicationSearch(appName string) *ApplicationSearch {
	return &ApplicationSearch{
		AppName:          appName,
		DeployClusters:   NewSet(),
		InstanceCount:    0,
		RegistryClusters: NewSet(),
	}
}

func (a *ApplicationSearch) MergeDataplane(dataplane *mesh.DataplaneResource) {
	a.InstanceCount++

	// merge inbounds
	inbounds := dataplane.Spec.Networking.Inbound
	for _, inbound := range inbounds {
		a.DeployClusters.Add(inbound.Tags[v1alpha1.ZoneTag])
	}
}

func (a *ApplicationSearch) GetRegistry(rt core_runtime.Runtime) {
	runtimeMode := rt.GetDeployMode()
	if runtimeMode == core.KubernetesMode {
		// In Kubernetes mode, registry cluster is the Kubernetes cluster itself
		a.RegistryClusters.Add(a.DeployClusters.Values()...)
	} else if runtimeMode == core.HalfHostMode || runtimeMode == core.UniversalMode {
		// In half or universal mode, registry cluster is the zookeeper cluster or other registry center
		registryURL := rt.RegistryCenter().GetURL()
		registryCluster := registryURL.GetParam(constants.RegistryClusterKey, "")
		a.RegistryClusters.Add(registryCluster)
	}
}
