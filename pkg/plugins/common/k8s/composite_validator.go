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
	"context"
)

import (
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type AdmissionValidator interface {
	webhook.AdmissionHandler
	InjectDecoder(d *admission.Decoder)
	Supports(admission.Request) bool
}

type CompositeValidator struct {
	Validators []AdmissionValidator
}

func (c *CompositeValidator) AddValidator(validator AdmissionValidator) {
	c.Validators = append(c.Validators, validator)
}

func (c *CompositeValidator) IntoWebhook(scheme *runtime.Scheme) *admission.Webhook {
	decoder := admission.NewDecoder(scheme)
	for _, validator := range c.Validators {
		validator.InjectDecoder(decoder)
	}

	return &admission.Webhook{
		Handler: admission.HandlerFunc(func(ctx context.Context, req admission.Request) admission.Response {
			for _, validator := range c.Validators {
				if validator.Supports(req) {
					resp := validator.Handle(ctx, req)
					if !resp.Allowed {
						return resp
					}
				}
			}
			return admission.Allowed("")
		}),
	}
}
