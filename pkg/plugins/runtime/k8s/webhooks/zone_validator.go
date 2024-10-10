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
	"net/http"
)

import (
	v1 "k8s.io/api/admission/v1"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core/managers/apis/zone"
	k8s_common "github.com/apache/dubbo-kubernetes/pkg/plugins/common/k8s"
	mesh_k8s "github.com/apache/dubbo-kubernetes/pkg/plugins/resources/k8s/native/api/v1alpha1"
)

func NewZoneValidatorWebhook(validator zone.Validator, unsafeDelete bool) k8s_common.AdmissionValidator {
	return &ZoneValidator{
		validator:    validator,
		unsafeDelete: unsafeDelete,
	}
}

type ZoneValidator struct {
	validator    zone.Validator
	unsafeDelete bool
}

func (z *ZoneValidator) InjectDecoder(_ admission.Decoder) {
}

func (z *ZoneValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	switch req.Operation {
	case v1.Delete:
		return z.ValidateDelete(ctx, req)
	}
	return admission.Allowed("")
}

func (z *ZoneValidator) ValidateDelete(ctx context.Context, req admission.Request) admission.Response {
	if !z.unsafeDelete {
		if err := z.validator.ValidateDelete(ctx, req.Name); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
	}
	return admission.Allowed("")
}

func (z *ZoneValidator) Supports(req admission.Request) bool {
	gvk := mesh_k8s.GroupVersion.WithKind("Zone")
	return req.Kind.Kind == gvk.Kind && req.Kind.Version == gvk.Version && req.Kind.Group == gvk.Group
}
