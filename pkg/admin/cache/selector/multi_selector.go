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
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

// MultiSelectors is an implement of Selector to combine multiple selectors, use NewMultiSelector to create it, and use Add to build it
type MultiSelectors struct {
	applicationNames []string
	serviceNames     []string
	serviceGroups    []string
	serviceVersions  []string
}

func NewMultiSelector() *MultiSelectors {
	return &MultiSelectors{
		applicationNames: make([]string, 0),
		serviceNames:     make([]string, 0),
		serviceGroups:    make([]string, 0),
		serviceVersions:  make([]string, 0),
	}
}

func (s *MultiSelectors) Add(selector Selector) *MultiSelectors {
	switch selector.(type) {
	case *ApplicationSelector:
		s.applicationNames = append(s.applicationNames, selector.(*ApplicationSelector).Name)
	case *ServiceSelector:
		s.serviceNames = append(s.serviceNames, selector.(*ServiceSelector).Name)
		if selector.(*ServiceSelector).Group != "" {
			s.serviceGroups = append(s.serviceGroups, selector.(*ServiceSelector).Group)
		}
		if selector.(*ServiceSelector).Version != "" {
			s.serviceVersions = append(s.serviceVersions, selector.(*ServiceSelector).Version)
		}
	}
	return s
}

func (s *MultiSelectors) AsLabelsSelector() labels.Selector {
	requirements := make([]labels.Requirement, 0)

	if len(s.applicationNames) > 0 {
		req, err := labels.NewRequirement(constant.ApplicationLabel, selection.In, s.applicationNames)
		if err != nil {
			logger.Errorf("failed to create requirement for application selector: %v", err)
		}
		requirements = append(requirements, *req)
	}

	if len(s.serviceNames) > 0 {
		req, err := labels.NewRequirement(constant.ServiceKeyLabel, selection.In, s.serviceNames)
		if err != nil {
			logger.Errorf("failed to create requirement for service selector: %v", err)
		}
		requirements = append(requirements, *req)
	}

	if len(s.serviceGroups) > 0 {
		req, err := labels.NewRequirement(constant.GroupLabel, selection.In, s.serviceGroups)
		if err != nil {
			logger.Errorf("failed to create requirement for group selector: %v", err)
		}
		requirements = append(requirements, *req)
	}

	if len(s.serviceVersions) > 0 {
		req, err := labels.NewRequirement(constant.VersionLabel, selection.In, s.serviceVersions)
		if err != nil {
			logger.Errorf("failed to create requirement for version selector: %v", err)
		}
		requirements = append(requirements, *req)
	}

	return labels.NewSelector().Add(requirements...)
}

func (s *MultiSelectors) ApplicationOptions() (Options, bool) {
	if len(s.applicationNames) == 0 {
		return nil, false
	}
	return newOptions(s.applicationNames...), true
}

func (s *MultiSelectors) ServiceNameOptions() (Options, bool) {
	if len(s.serviceNames) == 0 {
		return nil, false
	}
	return newOptions(s.serviceNames...), true
}

func (s *MultiSelectors) ServiceGroupOptions() (Options, bool) {
	if len(s.serviceGroups) == 0 {
		return nil, false
	}
	return newOptions(s.serviceGroups...), true
}

func (s *MultiSelectors) ServiceVersionOptions() (Options, bool) {
	if len(s.serviceVersions) == 0 {
		return nil, false
	}
	return newOptions(s.serviceVersions...), true
}
