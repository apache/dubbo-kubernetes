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

package unversioned

import (
	"encoding/json"
)

import (
	"google.golang.org/protobuf/proto"
)

import (
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model/rest/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/registry"
	util_proto "github.com/apache/dubbo-kubernetes/pkg/util/proto"
)

type Resource struct {
	Meta v1alpha1.ResourceMeta
	Spec core_model.ResourceSpec
}

func (r *Resource) GetMeta() v1alpha1.ResourceMeta {
	if r == nil {
		return v1alpha1.ResourceMeta{}
	}
	return r.Meta
}

func (r *Resource) GetSpec() core_model.ResourceSpec {
	if r == nil {
		return nil
	}
	return r.Spec
}

var (
	_ json.Marshaler   = &Resource{}
	_ json.Unmarshaler = &Resource{}
)

func (r *Resource) MarshalJSON() ([]byte, error) {
	var specBytes []byte
	if r.Spec != nil {
		bytes, err := core_model.ToJSON(r.Spec)
		if err != nil {
			return nil, err
		}
		specBytes = bytes
	}

	metaJSON, err := json.Marshal(r.Meta)
	if err != nil {
		return nil, err
	}

	if len(specBytes) == 0 || string(specBytes) == "{}" { // spec is nil or empty
		return metaJSON, nil
	} else {
		// remove the } of meta JSON, { of spec JSON and join it by ,
		return append(append(metaJSON[:len(metaJSON)-1], byte(',')), specBytes[1:]...), nil
	}
}

func (r *Resource) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &r.Meta); err != nil {
		return err
	}
	if r.Spec == nil {
		newR, err := registry.Global().NewObject(core_model.ResourceType(r.Meta.Type))
		if err != nil {
			return err
		}
		r.Spec = newR.GetSpec()
	}
	if err := util_proto.FromJSON(data, r.Spec.(proto.Message)); err != nil {
		return err
	}
	return nil
}

func (r *Resource) ToCore() (core_model.Resource, error) {
	resource, err := registry.Global().NewObject(core_model.ResourceType(r.Meta.Type))
	if err != nil {
		return nil, err
	}
	resource.SetMeta(&r.Meta)
	if err := resource.SetSpec(r.Spec); err != nil {
		return nil, err
	}
	return resource, nil
}
