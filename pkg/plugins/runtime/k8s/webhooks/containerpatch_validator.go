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
)

import (
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

import (
	k8s_common "github.com/apache/dubbo-kubernetes/pkg/plugins/common/k8s"
	mesh_k8s "github.com/apache/dubbo-kubernetes/pkg/plugins/resources/k8s/native/api/v1alpha1"
)

type ContainerPatchValidator struct {
	SystemNamespace string
}

func NewContainerPatchValidatorWebhook() k8s_common.AdmissionValidator {
	return &ContainerPatchValidator{}
}

func (h *ContainerPatchValidator) InjectDecoder(d *admission.Decoder) {
}

func (h *ContainerPatchValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	if req.Namespace != h.SystemNamespace {
		return admission.Denied("ContainerPatch can only be placed in " + h.SystemNamespace + " namespace. It can be however referenced by pods in all namespaces")
	}
	return admission.Allowed("")
}

func (h *ContainerPatchValidator) Supports(req admission.Request) bool {
	gvk := mesh_k8s.GroupVersion.WithKind("ContainerPatch")
	return req.Kind.Kind == gvk.Kind && req.Kind.Version == gvk.Version && req.Kind.Group == gvk.Group
}
