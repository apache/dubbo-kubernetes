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

package resp

import "github.com/apache/dubbo-kubernetes/pkg/admin/cache"

type ApplicationOverview struct {
	Name            string `json:"name"`
	InstanceCount   int    `json:"instanceCount"`
	DeployCluster   string `json:"deployCluster"`
	RegisterCluster string `json:"registerCluster"`
}

type ApplicationResp struct {
	Name      string          `json:"name"`
	Instances []*InstanceResp `json:"instances"`
	Services  []*ServiceResp  `json:"services"`
}

type InstanceResp struct {
	Name   string            `json:"name"`
	Ip     string            `json:"ip"`
	Port   string            `json:"port"`
	Status string            `json:"status"`
	Node   string            `json:"node"`
	Labels map[string]string `json:"labels"`
}

func (i *InstanceResp) FromCacheModel(instanceModel *cache.InstanceModel) *InstanceResp {
	i.Name = instanceModel.Name
	i.Ip = instanceModel.Ip
	i.Port = instanceModel.Port
	i.Status = instanceModel.Status
	i.Node = instanceModel.Node
	i.Labels = instanceModel.Labels
	return i
}

type ServiceResp struct {
	Category string            `json:"category"`
	Name     string            `json:"name"`
	Group    string            `json:"group"`
	Version  string            `json:"version"`
	Labels   map[string]string `json:"labels"`
}

func (s *ServiceResp) FromCacheModel(serviceModel *cache.ServiceModel) *ServiceResp {
	s.Category = serviceModel.Category
	s.Name = serviceModel.Name
	s.Group = serviceModel.Group
	s.Version = serviceModel.Version
	s.Labels = serviceModel.Labels
	return s
}
