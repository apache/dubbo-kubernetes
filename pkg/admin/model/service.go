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
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
)

type ServiceSearchReq struct {
	AppName string `form:"appName"`
	PageReq
}

type ServiceSearchResp struct {
	ServiceName   string         `json:"serviceName"`
	AvgQPS        string         `json:"avgQPS"`
	AvgRT         string         `json:"avgRT"`
	RequestTotal  string         `json:"RequestTotal"`
	VersionGroups []VersionGroup `json:"versionGroups"`
}

func (s *ServiceSearchResp) FromServiceDataplaneResource(dataplane *mesh.DataplaneResource) *ServiceSearchResp {
	// TODO: get real data
	s.ServiceName = dataplane.Meta.GetName()
	s.VersionGroups = make([]VersionGroup, 0)
	s.AvgQPS = "0.5"
	s.AvgRT = "345ms"
	s.RequestTotal = "1850"
	return s
}

type ServiceTabDistributionReq struct {
	AppName string `form:"appName"`
}

type ServiceTabDistributionResp struct {
	AppName      string `json:"appName"`
	InstanceName string `json:"instanceName"`
	Endpoint     string `json:"endpoint"`
	TimeOut      string `json:"timeOut"`
	Retries      string `json:"retries"`
}

func (s *ServiceTabDistributionResp) FromServiceDataplaneResource(dataplane *mesh.DataplaneResource) *ServiceTabDistributionResp {
	// TODO: get real data
	s.AppName = dataplane.GetMeta().GetLabels()[v1alpha1.AppTag]
	s.InstanceName = "instancedemo"
	s.Endpoint = "0.5"
	s.TimeOut = "345ms"
	s.Retries = "1850"
	return s
}

type VersionGroup struct {
	Version string `json:"version"`
	Group   string `json:"group"`
}
