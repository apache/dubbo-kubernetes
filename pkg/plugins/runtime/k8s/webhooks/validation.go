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
	"fmt"
	"net/http"
	"strings"
)

import (
	v1 "k8s.io/api/admission/v1"

	authenticationv1 "k8s.io/api/authentication/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kube_runtime "k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/config/core"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_registry "github.com/apache/dubbo-kubernetes/pkg/core/resources/registry"
	"github.com/apache/dubbo-kubernetes/pkg/core/validators"
	k8s_common "github.com/apache/dubbo-kubernetes/pkg/plugins/common/k8s"
	k8s_model "github.com/apache/dubbo-kubernetes/pkg/plugins/resources/k8s/native/pkg/model"
	k8s_registry "github.com/apache/dubbo-kubernetes/pkg/plugins/resources/k8s/native/pkg/registry"
	"github.com/apache/dubbo-kubernetes/pkg/version"
)

func NewValidatingWebhook(
	converter k8s_common.Converter,
	coreRegistry core_registry.TypeRegistry,
	k8sRegistry k8s_registry.TypeRegistry,
	mode core.CpMode,
	federatedZone bool,
	disableOriginLabelValidation bool,
) k8s_common.AdmissionValidator {
	return &validatingHandler{
		coreRegistry:                 coreRegistry,
		k8sRegistry:                  k8sRegistry,
		converter:                    converter,
		mode:                         mode,
		federatedZone:                federatedZone,
		disableOriginLabelValidation: disableOriginLabelValidation,
	}
}

type validatingHandler struct {
	coreRegistry                 core_registry.TypeRegistry
	k8sRegistry                  k8s_registry.TypeRegistry
	converter                    k8s_common.Converter
	decoder                      admission.Decoder
	mode                         core.CpMode
	federatedZone                bool
	disableOriginLabelValidation bool
}

func (h *validatingHandler) InjectDecoder(d admission.Decoder) {
	h.decoder = d
}

func (h *validatingHandler) Handle(ctx context.Context, req admission.Request) admission.Response {
	_, err := h.coreRegistry.DescriptorFor(core_model.ResourceType(req.Kind.Kind))
	if err != nil {
		// we only care about types in the registry for this handler
		return admission.Allowed("")
	}

	coreRes, k8sObj, err := h.decode(req)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if resp := h.isOperationAllowed(req.UserInfo, coreRes); !resp.Allowed {
		return resp
	}

	switch req.Operation {
	case v1.Delete:
		return admission.Allowed("")
	default:
		if err := h.validateLabels(coreRes.GetMeta()); err.HasViolations() {
			return convertValidationErrorOf(err, k8sObj, k8sObj.GetObjectMeta())
		}

		return admission.Allowed("")
	}
}

func (h *validatingHandler) decode(req admission.Request) (core_model.Resource, k8s_model.KubernetesObject, error) {
	coreRes, err := h.coreRegistry.NewObject(core_model.ResourceType(req.Kind.Kind))
	if err != nil {
		return nil, nil, err
	}
	k8sObj, err := h.k8sRegistry.NewObject(coreRes.GetSpec())
	if err != nil {
		return nil, nil, err
	}

	switch req.Operation {
	case v1.Delete:
		if err := h.decoder.DecodeRaw(req.OldObject, k8sObj); err != nil {
			return nil, nil, err
		}
	default:
		if err := h.decoder.Decode(req, k8sObj); err != nil {
			return nil, nil, err
		}
	}

	if err := h.converter.ToCoreResource(k8sObj, coreRes); err != nil {
		return nil, nil, err
	}
	return coreRes, k8sObj, nil
}

// Note that this func does not validate ConfigMap and Secret since this webhook does not support those
func (h *validatingHandler) isOperationAllowed(userInfo authenticationv1.UserInfo, r core_model.Resource) admission.Response {
	if !h.isResourceTypeAllowed(r.Descriptor()) {
		return resourceTypeIsNotAllowedResponse(r.Descriptor().Name, h.mode)
	}

	if !h.isResourceAllowed(r) {
		return resourceIsNotAllowedResponse()
	}

	return admission.Allowed("")
}

func (h *validatingHandler) isResourceTypeAllowed(d core_model.ResourceTypeDescriptor) bool {
	if d.DDSFlags == core_model.DDSDisabledFlag {
		return true
	}
	if h.mode == core.Global && !d.DDSFlags.Has(core_model.AllowedOnGlobalSelector) {
		return false
	}
	if h.federatedZone && !d.DDSFlags.Has(core_model.AllowedOnZoneSelector) {
		return false
	}
	return true
}

func (h *validatingHandler) isResourceAllowed(r core_model.Resource) bool {
	if !h.federatedZone || !r.Descriptor().IsPluginOriginated {
		return true
	}
	if !h.disableOriginLabelValidation {
		if origin, ok := core_model.ResourceOrigin(r.GetMeta()); !ok || origin != mesh_proto.ZoneResourceOrigin {
			return false
		}
	}
	return true
}

func (h *validatingHandler) validateLabels(rm core_model.ResourceMeta) validators.ValidationError {
	var verr validators.ValidationError
	if origin, ok := core_model.ResourceOrigin(rm); ok {
		if err := origin.IsValid(); err != nil {
			verr.AddViolationAt(validators.Root().Field("labels").Key(mesh_proto.ResourceOriginLabel), err.Error())
		}
	}
	return verr
}

func resourceIsNotAllowedResponse() admission.Response {
	return admission.Response{
		AdmissionResponse: v1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Status:  "Failure",
				Message: fmt.Sprintf("Operation not allowed. Applying policies on Zone CP requires '%s' label to be set to '%s'.", mesh_proto.ResourceOriginLabel, mesh_proto.ZoneResourceOrigin),
				Reason:  "Forbidden",
				Code:    403,
				Details: &metav1.StatusDetails{
					Causes: []metav1.StatusCause{
						{
							Type:    "FieldValueInvalid",
							Message: "cannot be empty",
							Field:   "metadata.labels[dubbo.io/origin]",
						},
					},
				},
			},
		},
	}
}

func resourceTypeIsNotAllowedResponse(resType core_model.ResourceType, cpMode core.CpMode) admission.Response {
	otherCpMode := ""
	if cpMode == core.Zone {
		otherCpMode = core.Global
	} else if cpMode == core.Global {
		otherCpMode = core.Zone
	}
	return admission.Response{
		AdmissionResponse: v1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Status: "Failure",
				Message: fmt.Sprintf("Operation not allowed. %s resources like %s can be updated or deleted only "+
					"from the %s control plane and not from a %s control plane.", version.Product, resType, strings.ToUpper(otherCpMode), strings.ToUpper(cpMode)),
				Reason: "Forbidden",
				Code:   403,
				Details: &metav1.StatusDetails{
					Causes: []metav1.StatusCause{
						{
							Type:    "FieldValueInvalid",
							Message: "cannot be empty",
							Field:   "metadata.annotations[dubbo.io/synced]",
						},
					},
				},
			},
		},
	}
}

func (h *validatingHandler) Supports(admission.Request) bool {
	return true
}

func convertValidationErrorOf(dubboErr validators.ValidationError, obj kube_runtime.Object, objMeta metav1.Object) admission.Response {
	details := &metav1.StatusDetails{
		Name: objMeta.GetName(),
		Kind: obj.GetObjectKind().GroupVersionKind().Kind,
	}
	resp := admission.Response{
		AdmissionResponse: v1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Status:  "Failure",
				Message: dubboErr.Error(),
				Reason:  "Invalid",
				Code:    int32(422),
				Details: details,
			},
		},
	}
	for _, violation := range dubboErr.Violations {
		cause := metav1.StatusCause{
			Type:    "FieldValueInvalid",
			Message: violation.Message,
			Field:   violation.Field,
		}
		details.Causes = append(details.Causes, cause)
	}
	return resp
}
