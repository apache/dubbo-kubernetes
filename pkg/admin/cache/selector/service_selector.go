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

package selector

import (
	"github.com/apache/dubbo-kubernetes/pkg/admin/constant"
	"k8s.io/apimachinery/pkg/labels"
)

type ServiceSelector struct {
	Name    string
	Group   string
	Version string
}

func NewServiceSelector(name, group, version string) *ServiceSelector {
	return &ServiceSelector{
		Name:    name,
		Group:   group,
		Version: version,
	}
}

func (s *ServiceSelector) AsLabelsSelector() labels.Selector {
	selector := labels.Set{
		constant.ServiceKeyLabel: s.Name,
	}
	if s.Group != "" {
		selector[constant.GroupLabel] = s.Group
	}
	if s.Version != "" {
		selector[constant.VersionLabel] = s.Version
	}
	return selector.AsSelector()
}

func (s *ServiceSelector) ApplicationOptions() (Options, bool) {
	return nil, false
}

func (s *ServiceSelector) ServiceNameOptions() (Options, bool) {
	return newOptions(s.Name), true
}

func (s *ServiceSelector) ServiceGroupOptions() (Options, bool) {
	return newOptions(s.Group), true
}

func (s *ServiceSelector) ServiceVersionOptions() (Options, bool) {
	return newOptions(s.Version), true
}
