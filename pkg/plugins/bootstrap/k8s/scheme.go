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
	"github.com/pkg/errors"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	kube_runtime "k8s.io/apimachinery/pkg/runtime"

	kube_client_scheme "k8s.io/client-go/kubernetes/scheme"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/plugins/policies"
	mesh_k8s "github.com/apache/dubbo-kubernetes/pkg/plugins/resources/k8s/native/api/v1alpha1"
)

// NewScheme creates a new scheme with all the necessary schemas added already (dubbo CRD, builtin resources).
func NewScheme() (*kube_runtime.Scheme, error) {
	s := kube_runtime.NewScheme()
	if err := kube_client_scheme.AddToScheme(s); err != nil {
		return nil, errors.Wrapf(err, "could not add client resources to scheme")
	}
	if err := mesh_k8s.AddToScheme(s); err != nil {
		return nil, errors.Wrapf(err, "could not add %q to scheme", mesh_k8s.GroupVersion)
	}
	if err := apiextensionsv1.AddToScheme(s); err != nil {
		return nil, errors.Wrapf(err, "could not add %q to scheme", apiextensionsv1.SchemeGroupVersion)
	}
	if err := policies.AddToScheme(s); err != nil {
		return nil, err
	}
	return s, nil
}
