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

package servicesV2

import (
	"github.com/apache/dubbo-kubernetes/pkg/admin/cache"
	"github.com/apache/dubbo-kubernetes/pkg/admin/cache/selector"
	"github.com/apache/dubbo-kubernetes/pkg/admin/model/resp"
)

type ResourceService interface {
	FindApplications(namespace string) ([]string, error)
	FindApplicationDetail(namespace string, application string) (*resp.ApplicationResp, error)
}

func NewResourceService() ResourceService {
	return &resourceServiceImpl{}
}

type resourceServiceImpl struct{}

func (p *resourceServiceImpl) FindApplications(namespace string) ([]string, error) {
	appModels, err := cache.GetCache().GetApplications(namespace)
	if err != nil {
		return nil, err
	}
	var apps []string
	for _, app := range appModels {
		apps = append(apps, app.Name)
	}
	return apps, nil
}

func (p *resourceServiceImpl) FindApplicationDetail(namespace string, application string) (*resp.ApplicationResp, error) {
	serviceModels, err := cache.GetCache().GetServicesWithSelector(namespace, selector.NewApplicationSelector(application))
	if err != nil {
		return nil, err
	}
	serviceRespSlice := make([]*resp.ServiceResp, 0, len(serviceModels))
	for _, service := range serviceModels {
		item := &resp.ServiceResp{}
		item.FromCacheModel(service)
		serviceRespSlice = append(serviceRespSlice, item)
	}

	instanceModels, err := cache.GetCache().GetInstancesWithSelector(namespace, selector.NewApplicationSelector(application))
	if err != nil {
		return nil, err
	}
	instanceRespSlice := make([]*resp.InstanceResp, 0, len(instanceModels))
	for _, instance := range instanceModels {
		item := &resp.InstanceResp{}
		item.FromCacheModel(instance)
		instanceRespSlice = append(instanceRespSlice, item)
	}

	return &resp.ApplicationResp{
		Name:      application,
		Services:  serviceRespSlice,
		Instances: instanceRespSlice,
	}, nil
}
