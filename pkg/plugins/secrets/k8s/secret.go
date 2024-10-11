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
	"fmt"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/wrapperspb"
	v1 "k8s.io/api/core/v1"
	k8s "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	system_proto "github.com/apache/dubbo-kubernetes/api/system/v1alpha1"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/resources/k8s/native/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/runtime/k8s/metadata"
)

// Secret is a KubernetesObject for Dubbo's Secret and GlobalSecret.
// Note that it's not registered in TypeRegistry because we cannot multiply KubernetesObject
// for a single Spec (both Secret and GlobalSecret has same Spec).
type Secret struct {
	v1.Secret
}

func NewSecret(typ v1.SecretType) *Secret {
	return &Secret{
		Secret: v1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1.SchemeGroupVersion.String(),
				Kind:       "Secret",
			},
			Type: typ,
		},
	}
}

var _ model.KubernetesObject = &Secret{}

func (s *Secret) GetObjectMeta() *k8s.ObjectMeta {
	return &s.ObjectMeta
}

func (s *Secret) SetObjectMeta(meta *k8s.ObjectMeta) {
	s.ObjectMeta = *meta
}

func (s *Secret) GetMesh() string {
	if mesh, ok := s.ObjectMeta.Labels[metadata.DubboMeshLabel]; ok {
		return mesh
	} else {
		return core_model.DefaultMesh
	}
}

func (s *Secret) SetMesh(mesh string) {
	if s.ObjectMeta.Labels == nil {
		s.ObjectMeta.Labels = map[string]string{}
	}
	s.ObjectMeta.Labels[metadata.DubboMeshLabel] = mesh
}

func (s *Secret) GetSpec() (core_model.ResourceSpec, error) {
	bytes, ok := s.Data["value"]
	if !ok {
		return nil, nil
	}
	return &system_proto.Secret{
		Data: &wrapperspb.BytesValue{
			Value: bytes,
		},
	}, nil
}

func (s *Secret) SetSpec(spec core_model.ResourceSpec) {
	if _, ok := spec.(*system_proto.Secret); !ok {
		panic(fmt.Sprintf("unexpected protobuf message type %T", spec))
	}
	s.Data = map[string][]byte{
		"value": spec.(*system_proto.Secret).GetData().GetValue(),
	}
}

func (s *Secret) GetStatus() (core_model.ResourceStatus, error) {
	return nil, nil
}

func (s *Secret) SetStatus(status core_model.ResourceStatus) error {
	return errors.New("status not supported")
}

func (s *Secret) Scope() model.Scope {
	return model.ScopeNamespace
}
