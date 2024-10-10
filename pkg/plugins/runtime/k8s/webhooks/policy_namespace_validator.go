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
)

import (
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

import (
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/registry"
)

type PolicyNamespaceValidator struct {
	Decoder         admission.Decoder
	SystemNamespace string
}

func (p *PolicyNamespaceValidator) InjectDecoder(decoder admission.Decoder) {
	p.Decoder = decoder
}

func (p *PolicyNamespaceValidator) Handle(ctx context.Context, request admission.Request) admission.Response {
	if request.Namespace != p.SystemNamespace {
		return admission.Denied(fmt.Sprintf("policy can only be created in the system namespace:%s", p.SystemNamespace))
	}
	return admission.Allowed("")
}

func (p *PolicyNamespaceValidator) Supports(request admission.Request) bool {
	desc, err := registry.Global().DescriptorFor(core_model.ResourceType(request.Kind.Kind))
	if err != nil {
		return false
	}
	return desc.IsPluginOriginated
}
