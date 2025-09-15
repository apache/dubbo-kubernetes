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

package gvr

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	CustomResourceDefinition       = schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}
	MutatingWebhookConfiguration   = schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "MutatingWebhookConfiguration"}
	ValidatingWebhookConfiguration = schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "ValidatingWebhookConfiguration"}
	Deployment                     = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	StatefulSet                    = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}
	DaemonSet                      = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}
	Job                            = schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"}
	Namespace                      = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}
	ConfigMap                      = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	Secret                         = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}
	Service                        = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
	ServiceAccount                 = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"}
	MeshConfig                     = schema.GroupVersionResource{Group: "", Version: "v1alpha1", Resource: "meshconfigs"}
)

func IsClusterScoped(g schema.GroupVersionResource) bool {
	switch g {
	case ConfigMap:
		return false
	case Namespace:
		return true
	case DaemonSet:
		return false
	case Deployment:
		return false
	case StatefulSet:
		return false
	case Secret:
		return false
	case Service:
		return false
	case ServiceAccount:
		return false
	}
	return false
}
