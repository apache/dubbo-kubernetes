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

package registry

import (
	"strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/registry"

	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
)

type MetadataInstance struct {
	registry.ServiceInstance
}

func (m *MetadataInstance) GetServiceName() string {
	serviceName := m.ServiceInstance.GetServiceName()
	// remove the group prefix from name of nacos instance service
	groupServiceName := strings.SplitN(serviceName, constant.CONFIG_INFO_SPLITER, 2)
	if len(groupServiceName) > 1 {
		serviceName = groupServiceName[1]
	}
	return serviceName
}

func ConvertToMetadataInstance(serviceInstance registry.ServiceInstance) *MetadataInstance {
	return &MetadataInstance{
		ServiceInstance: serviceInstance,
	}
}
