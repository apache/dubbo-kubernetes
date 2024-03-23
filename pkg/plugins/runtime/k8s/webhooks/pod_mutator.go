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
	kube_core "k8s.io/api/core/v1"

	kube_webhook "sigs.k8s.io/controller-runtime/pkg/webhook"
	kube_admission "sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type PodMutator func(context.Context, *kube_core.Pod) error

func PodMutatingWebhook(mutator PodMutator) *kube_admission.Webhook {
	return &kube_admission.Webhook{
		Handler: &podMutatingHandler{mutator: mutator},
	}
}

type podMutatingHandler struct {
	mutator PodMutator
}

func (h *podMutatingHandler) Handle(ctx context.Context, req kube_webhook.AdmissionRequest) kube_webhook.AdmissionResponse {
	var pod kube_core.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		return kube_admission.Errored(http.StatusBadRequest, err)
	}
	pod.Namespace = req.Namespace
	if err := h.mutator(ctx, &pod); err != nil {
		return kube_admission.Errored(http.StatusInternalServerError, err)
	}
	mutatedRaw, err := json.Marshal(pod)
	if err != nil {
		return kube_admission.Errored(http.StatusInternalServerError, err)
	}
	return kube_admission.PatchResponseFromRaw(req.Object.Raw, mutatedRaw)
}
