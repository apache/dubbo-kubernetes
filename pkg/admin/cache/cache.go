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

package cache

import (
	"github.com/apache/dubbo-kubernetes/pkg/admin/cache/selector"
)

var cache Cache

func SetCache(c Cache) {
	cache = c
}

func GetCache() Cache {
	return cache
}

type Cache interface {
	GetApplications(namespace string) ([]*ApplicationModel, error)
	GetApplicationsWithSelector(namespace string, selector selector.Selector) ([]*ApplicationModel, error)
	GetWorkloads(namespace string) ([]*WorkloadModel, error)
	GetWorkloadsWithSelector(namespace string, selector selector.Selector) ([]*WorkloadModel, error)
	GetInstances(namespace string) ([]*InstanceModel, error)
	GetInstancesWithSelector(namespace string, selector selector.Selector) ([]*InstanceModel, error)
	GetServices(namespace string) ([]*ServiceModel, error)
	GetServicesWithSelector(namespace string, selector selector.Selector) ([]*ServiceModel, error)

	// TODO: add support for other resources
	// Telemetry/Metrics
	// ConditionRule
	// TagRule
	// DynamicConfigurationRule
}

type ApplicationModel struct {
	Name string
}

type WorkloadModel struct {
	Application *ApplicationModel
	Name        string
	Type        string
	Image       string
	Labels      map[string]string
}

type InstanceModel struct {
	Application *ApplicationModel
	Workload    *WorkloadModel
	Name        string
	Ip          string
	Port        string
	Status      string
	Node        string
	Labels      map[string]string
}

type ServiceModel struct {
	Application *ApplicationModel
	Category    string
	Name        string
	Labels      map[string]string
	ServiceKey  string
	Group       string
	Version     string
}
