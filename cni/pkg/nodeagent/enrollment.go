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

package nodeagent

import (
	"strings"

	cniconstants "github.com/apache/dubbo-kubernetes/cni/pkg/constants"
	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
	"github.com/dubml/api/annotation"
	"github.com/dubml/api/label"
	corev1 "k8s.io/api/core/v1"
)

const legacyInjectionLabel = "dubbo-injection"

// IsEnrolledPod returns true when a Pod should be intercepted and managed by
// the node-level CNI Triple tunnel agent.
func IsEnrolledPod(pod *corev1.Pod, ns *corev1.Namespace, systemNamespace string) bool {
	if pod == nil || pod.DeletionTimestamp != nil {
		return false
	}
	if pod.Spec.HostNetwork {
		return false
	}
	if systemNamespace == "" {
		systemNamespace = "dubbo-system"
	}
	if pod.Namespace == systemNamespace || pod.Namespace == "kube-system" {
		return false
	}

	// Explicit opt-out on Pod annotations or labels wins.
	if strings.EqualFold(pod.Annotations[cniconstants.RedirectionAnnotation], cniconstants.RedirectionDisabledValue) {
		return false
	}
	if strings.EqualFold(pod.Annotations[annotation.OrgApacheDubboInherentInject.Name], "false") {
		return false
	}
	if strings.EqualFold(pod.Labels[label.OrgApacheDubboInherentInject.Name], "false") {
		return false
	}

	// Explicit opt-in on Pod.
	if strings.EqualFold(pod.Annotations[cniconstants.RedirectionAnnotation], cniconstants.RedirectionEnabledValue) {
		return true
	}
	if containsCSV(pod.Annotations[annotation.OrgApacheDubboInjectTemplates.Name], inject.InherentTripleTemplateName) {
		return true
	}
	if strings.EqualFold(pod.Annotations[annotation.OrgApacheDubboInherentInject.Name], "true") ||
		strings.EqualFold(pod.Labels[label.OrgApacheDubboInherentInject.Name], "true") {
		return true
	}

	// Namespace-level enrollment for zero-modification arbitrary-language workloads.
	if ns != nil {
		if strings.EqualFold(ns.Labels[label.OrgApacheDubboInherentInject.Name], "true") ||
			strings.EqualFold(ns.Labels[legacyInjectionLabel], "enabled") {
			return true
		}
	}
	return false
}

func containsCSV(csv, target string) bool {
	for _, item := range strings.Split(csv, ",") {
		if strings.TrimSpace(item) == target {
			return true
		}
	}
	return false
}
