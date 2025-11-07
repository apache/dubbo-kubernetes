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

package kube

import (
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"
	kubeApiAdmissionv1beta1 "k8s.io/api/admission/v1beta1"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// APIVersion constants
	admissionAPIV1      = "admission.k8s.io/v1"
	admissionAPIV1beta1 = "admission.k8s.io/v1beta1"

	// Operation constants
	Create  string = "CREATE"
	Update  string = "UPDATE"
	Delete  string = "DELETE"
	Connect string = "CONNECT"
)

type AdmissionResponse struct {
	UID              types.UID         `json:"uid"`
	Allowed          bool              `json:"allowed"`
	Result           *metav1.Status    `json:"status,omitempty"`
	Patch            []byte            `json:"patch,omitempty"`
	PatchType        *string           `json:"patchType,omitempty"`
	AuditAnnotations map[string]string `json:"auditAnnotations,omitempty"`
	Warnings         []string          `json:"warnings,omitempty"`
}

type AdmissionRequest struct {
	UID                types.UID                    `json:"uid"`
	Kind               metav1.GroupVersionKind      `json:"kind"`
	Resource           metav1.GroupVersionResource  `json:"resource"`
	SubResource        string                       `json:"subResource,omitempty"`
	RequestSubResource string                       `json:"requestSubResource,omitempty"`
	RequestKind        *metav1.GroupVersionKind     `json:"requestKind,omitempty"`
	RequestResource    *metav1.GroupVersionResource `json:"requestResource,omitempty"`
	UserInfo           authenticationv1.UserInfo    `json:"userInfo"`
	Name               string                       `json:"name,omitempty"`
	Namespace          string                       `json:"namespace,omitempty"`
	Operation          string                       `json:"operation"`
	Object             runtime.RawExtension         `json:"object,omitempty"`
	OldObject          runtime.RawExtension         `json:"oldObject,omitempty"`
	DryRun             *bool                        `json:"dryRun,omitempty"`
	Options            runtime.RawExtension         `json:"options,omitempty"`
}

type AdmissionReview struct {
	metav1.TypeMeta
	Request  *AdmissionRequest  `json:"request,omitempty"`
	Response *AdmissionResponse `json:"response,omitempty"`
}

func AdmissionReviewKubeToAdapter(object runtime.Object) (*AdmissionReview, error) {
	var typeMeta metav1.TypeMeta
	var req *AdmissionRequest
	var resp *AdmissionResponse
	switch obj := object.(type) {
	case *kubeApiAdmissionv1beta1.AdmissionReview:
		typeMeta = obj.TypeMeta
		arv1beta1Response := obj.Response
		arv1beta1Request := obj.Request
		if arv1beta1Response != nil {
			resp = &AdmissionResponse{
				UID:      arv1beta1Response.UID,
				Allowed:  arv1beta1Response.Allowed,
				Result:   arv1beta1Response.Result,
				Patch:    arv1beta1Response.Patch,
				Warnings: arv1beta1Response.Warnings,
			}
			if arv1beta1Response.PatchType != nil {
				patchType := string(*arv1beta1Response.PatchType)
				resp.PatchType = &patchType
			}
		}
		if arv1beta1Request != nil {
			req = &AdmissionRequest{
				UID:       arv1beta1Request.UID,
				Kind:      arv1beta1Request.Kind,
				Resource:  arv1beta1Request.Resource,
				UserInfo:  arv1beta1Request.UserInfo,
				Name:      arv1beta1Request.Name,
				Namespace: arv1beta1Request.Namespace,
				Operation: string(arv1beta1Request.Operation),
				Object:    arv1beta1Request.Object,
				OldObject: arv1beta1Request.OldObject,
				DryRun:    arv1beta1Request.DryRun,
			}
		}

	case *admissionv1.AdmissionReview:
		typeMeta = obj.TypeMeta
		arv1Response := obj.Response
		arv1Request := obj.Request
		if arv1Response != nil {
			resp = &AdmissionResponse{
				UID:      arv1Response.UID,
				Allowed:  arv1Response.Allowed,
				Result:   arv1Response.Result,
				Patch:    arv1Response.Patch,
				Warnings: arv1Response.Warnings,
			}
			if arv1Response.PatchType != nil {
				patchType := string(*arv1Response.PatchType)
				resp.PatchType = &patchType
			}
		}

		if arv1Request != nil {
			req = &AdmissionRequest{
				UID:       arv1Request.UID,
				Kind:      arv1Request.Kind,
				Resource:  arv1Request.Resource,
				UserInfo:  arv1Request.UserInfo,
				Name:      arv1Request.Name,
				Namespace: arv1Request.Namespace,
				Operation: string(arv1Request.Operation),
				Object:    arv1Request.Object,
				OldObject: arv1Request.OldObject,
				DryRun:    arv1Request.DryRun,
			}
		}

	default:
		return nil, fmt.Errorf("unsupported type :%v", object.GetObjectKind())
	}

	return &AdmissionReview{
		TypeMeta: typeMeta,
		Request:  req,
		Response: resp,
	}, nil
}

func AdmissionReviewAdapterToKube(ar *AdmissionReview, apiVersion string) runtime.Object {
	var res runtime.Object
	arRequest := ar.Request
	arResponse := ar.Response
	if apiVersion == "" {
		apiVersion = admissionAPIV1beta1
	}
	switch apiVersion {
	case admissionAPIV1beta1:
		arv1beta1 := kubeApiAdmissionv1beta1.AdmissionReview{}
		if arRequest != nil {
			arv1beta1.Request = &kubeApiAdmissionv1beta1.AdmissionRequest{
				UID:                arRequest.UID,
				Kind:               arRequest.Kind,
				Resource:           arRequest.Resource,
				SubResource:        arRequest.SubResource,
				Name:               arRequest.Name,
				Namespace:          arRequest.Namespace,
				RequestKind:        arRequest.RequestKind,
				RequestResource:    arRequest.RequestResource,
				RequestSubResource: arRequest.RequestSubResource,
				Operation:          kubeApiAdmissionv1beta1.Operation(arRequest.Operation),
				UserInfo:           arRequest.UserInfo,
				Object:             arRequest.Object,
				OldObject:          arRequest.OldObject,
				DryRun:             arRequest.DryRun,
				Options:            arRequest.Options,
			}
		}
		if arResponse != nil {
			var patchType *kubeApiAdmissionv1beta1.PatchType
			if arResponse.PatchType != nil {
				patchType = (*kubeApiAdmissionv1beta1.PatchType)(arResponse.PatchType)
			}
			arv1beta1.Response = &kubeApiAdmissionv1beta1.AdmissionResponse{
				UID:              arResponse.UID,
				Allowed:          arResponse.Allowed,
				Result:           arResponse.Result,
				Patch:            arResponse.Patch,
				PatchType:        patchType,
				AuditAnnotations: arResponse.AuditAnnotations,
				Warnings:         arResponse.Warnings,
			}
		}
		arv1beta1.TypeMeta = ar.TypeMeta
		res = &arv1beta1
	case admissionAPIV1:
		arv1 := admissionv1.AdmissionReview{}
		if arRequest != nil {
			arv1.Request = &admissionv1.AdmissionRequest{
				UID:                arRequest.UID,
				Kind:               arRequest.Kind,
				Resource:           arRequest.Resource,
				SubResource:        arRequest.SubResource,
				Name:               arRequest.Name,
				Namespace:          arRequest.Namespace,
				RequestKind:        arRequest.RequestKind,
				RequestResource:    arRequest.RequestResource,
				RequestSubResource: arRequest.RequestSubResource,
				Operation:          admissionv1.Operation(arRequest.Operation),
				UserInfo:           arRequest.UserInfo,
				Object:             arRequest.Object,
				OldObject:          arRequest.OldObject,
				DryRun:             arRequest.DryRun,
				Options:            arRequest.Options,
			}
		}
		if arResponse != nil {
			var patchType *admissionv1.PatchType
			if arResponse.PatchType != nil {
				patchType = (*admissionv1.PatchType)(arResponse.PatchType)
			}
			arv1.Response = &admissionv1.AdmissionResponse{
				UID:              arResponse.UID,
				Allowed:          arResponse.Allowed,
				Result:           arResponse.Result,
				Patch:            arResponse.Patch,
				PatchType:        patchType,
				AuditAnnotations: arResponse.AuditAnnotations,
				Warnings:         arResponse.Warnings,
			}
		}
		arv1.TypeMeta = ar.TypeMeta
		res = &arv1
	}
	return res
}
