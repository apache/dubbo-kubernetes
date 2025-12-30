//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kstatus

import (
	"github.com/apache/dubbo-kubernetes/pkg/slices"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	StatusTrue  = "True"
	StatusFalse = "False"
)

func UpdateConditionIfChanged(conditions []metav1.Condition, condition metav1.Condition) []metav1.Condition {
	ret := slices.Clone(conditions)
	existing := slices.FindFunc(ret, func(cond metav1.Condition) bool {
		return cond.Type == condition.Type
	})
	if existing == nil {
		ret = append(ret, condition)
		return ret
	}

	if existing.Status == condition.Status {
		if existing.Message == condition.Message &&
			existing.ObservedGeneration == condition.ObservedGeneration {
			return conditions
		}
		condition.LastTransitionTime = existing.LastTransitionTime
	}
	*existing = condition

	return ret
}
