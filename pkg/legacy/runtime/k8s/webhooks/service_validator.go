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

	kube_core "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/validators"
)

// ServiceValidator validates Dubbo-specific annotations on Services.
type ServiceValidator struct {
	Decoder admission.Decoder
}

// Handle admits a Service only if Dubbo-specific annotations have proper values.
func (v *ServiceValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	svc := &kube_core.Service{}

	err := v.Decoder.Decode(req, svc)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if err := v.validate(svc); err != nil {
		if verr, ok := err.(*validators.ValidationError); ok {
			return convertValidationErrorOf(*verr, svc, svc)
		}
		return admission.Denied(err.Error())
	}

	return admission.Allowed("")
}

func (v *ServiceValidator) validate(svc *kube_core.Service) error {
	verr := &validators.ValidationError{}
	for _, svcPort := range svc.Spec.Ports {
		protocolAnnotation := fmt.Sprintf("%d.service.dubbo.io/protocol", svcPort.Port)
		protocolAnnotationValue, exists := svc.Annotations[protocolAnnotation]
		if exists && core_mesh.ParseProtocol(protocolAnnotationValue) == core_mesh.ProtocolUnknown {
			verr.AddViolationAt(validators.RootedAt("metadata").Field("annotations").Key(protocolAnnotation),
				fmt.Sprintf("value %q is not valid. %s", protocolAnnotationValue, core_mesh.AllowedValuesHint(core_mesh.SupportedProtocols.Strings()...)))
		}
	}
	return verr.OrNil()
}
