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
	"github.com/apache/dubbo-kubernetes/pkg/admin/errors"
	"github.com/apache/dubbo-kubernetes/pkg/admin/model/resp"
)

type ResourceService interface {
	SearchApplications(namespace string, keywords string) ([]*resp.ApplicationOverview, error)
}

func NewResourceService() ResourceService {
	return &resourceServiceImpl{}
}

type resourceServiceImpl struct{}

func (s *resourceServiceImpl) SearchApplications(namespace string, keywords string) ([]*resp.ApplicationOverview, error) {
	var appModels []*cache.ApplicationModel
	var err error
	if keywords == "" {
		appModels, err = cache.GetCache().GetApplications(namespace)
	} else {
		appModels, err = cache.GetCache().GetApplicationsWithSelector(namespace, selector.NewApplicationSelector(keywords))
	}
	if err != nil {
		return nil, errors.NewBizErrorWithStack(resp.CoreCacheErrorCode, err)
	}

	var appOverviews []*resp.ApplicationOverview
	for _, app := range appModels {
		appName := app.Name
		instanceModels, err := cache.GetCache().GetInstancesWithSelector(namespace, selector.NewApplicationSelector(appName)) // TODO: create a new method to get instance count directly
		if err != nil {
			return nil, errors.NewBizErrorWithStack(resp.CoreCacheErrorCode, err)
		}
		appOverviews = append(appOverviews, &resp.ApplicationOverview{
			Name:            appName,
			InstanceCount:   len(instanceModels),
			DeployCluster:   "", // TODO: add support for deploy cluster
			RegisterCluster: "", // TODO: add support for register cluster
		})
	}

	return appOverviews, nil
}
