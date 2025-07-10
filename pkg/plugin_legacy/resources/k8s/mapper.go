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

package k8s

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	k8s_model "github.com/apache/dubbo-kubernetes/pkg/plugins/resources/k8s/native/pkg/model"
)

type ResourceMapperFunc func(resource model.Resource, namespace string) (k8s_model.KubernetesObject, error)

// NewKubernetesMapper creates a ResourceMapper that returns the k8s object as is. This is meant to be used when the underlying store is kubernetes
func NewKubernetesMapper(kubeFactory KubeFactory) ResourceMapperFunc {
	return func(resource model.Resource, namespace string) (k8s_model.KubernetesObject, error) {
		res, err := (&SimpleConverter{KubeFactory: kubeFactory}).ToKubernetesObject(resource)
		if err != nil {
			return nil, err
		}
		if namespace != "" {
			res.SetNamespace(namespace)
		}
		return res, err
	}
}

// NewInferenceMapper creates a ResourceMapper that infers a k8s resource from the core_model. Extract namespace from the name if necessary.
// This mostly useful when the underlying store is not kubernetes but you want to show what a kubernetes version of the policy would be like (in global for example).
func NewInferenceMapper(systemNamespace string, kubeFactory KubeFactory) ResourceMapperFunc {
	return func(resource model.Resource, namespace string) (k8s_model.KubernetesObject, error) {
		rs, err := kubeFactory.NewObject(resource)
		if err != nil {
			return nil, err
		}
		if rs.Scope() == k8s_model.ScopeNamespace {
			if namespace != "" { // If the user is forcing the namespace accept it.
				rs.SetNamespace(namespace)
			} else {
				rs.SetNamespace(systemNamespace)
			}
		}
		rs.SetName(resource.GetMeta().GetName())
		rs.SetMesh(resource.GetMeta().GetMesh())
		rs.SetCreationTimestamp(v1.NewTime(resource.GetMeta().GetCreationTime()))
		rs.SetSpec(resource.GetSpec())
		return rs, nil
	}
}
