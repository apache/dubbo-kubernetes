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

package webhooks

import (
	"context"
	"encoding/json"
	"net/http"
)

import (
	kube_runtime "k8s.io/apimachinery/pkg/runtime"

	kube_client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

import (
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_registry "github.com/apache/dubbo-kubernetes/pkg/core/resources/registry"
	k8s_model "github.com/apache/dubbo-kubernetes/pkg/plugins/resources/k8s/native/pkg/model"
	k8s_registry "github.com/apache/dubbo-kubernetes/pkg/plugins/resources/k8s/native/pkg/registry"
)

type OwnerReferenceMutator struct {
	Client       kube_client.Client
	CoreRegistry core_registry.TypeRegistry
	K8sRegistry  k8s_registry.TypeRegistry
	Decoder      *admission.Decoder
	Scheme       *kube_runtime.Scheme
}

func (m *OwnerReferenceMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	resType := core_model.ResourceType(req.Kind.Kind)

	coreRes, err := m.CoreRegistry.NewObject(resType)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	obj, err := m.K8sRegistry.NewObject(coreRes.GetSpec())
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// unmarshal k8s object from the request
	if err := m.Decoder.Decode(req, obj); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	var owner k8s_model.KubernetesObject

	// TODO
	if err := controllerutil.SetOwnerReference(owner, obj, m.Scheme); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	mutatedRaw, err := json.Marshal(obj)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, mutatedRaw)
}
