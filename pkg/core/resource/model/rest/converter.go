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

package rest

import (
	"github.com/apache/dubbo-kubernetes/pkg/core/resource/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resource/model/rest/unversioned"
	"github.com/apache/dubbo-kubernetes/pkg/core/resource/model/rest/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/resource/registry"
)

var From = &from{}

type from struct{}

func (f *from) Resource(r model.Resource) Resource {
	if r == nil {
		return nil
	}

	meta := f.Meta(r)
	if r.Descriptor().IsPluginOriginated {
		return &v1alpha1.Resource{
			ResourceMeta: meta,
			Spec:         r.GetSpec(),
		}
	} else {
		return &unversioned.Resource{
			Meta: meta,
			Spec: r.GetSpec(),
		}
	}
}

func (f *from) Meta(r model.Resource) v1alpha1.ResourceMeta {
	meta := v1alpha1.ResourceMeta{}
	if r == nil {
		return meta
	}
	if r.GetMeta() != nil {
		var meshName string
		if r.Descriptor().Scope == model.ScopeMesh {
			meshName = r.GetMeta().GetMesh()
		}
		meta = v1alpha1.ResourceMeta{
			Mesh:             meshName,
			Type:             string(r.Descriptor().Name),
			Name:             r.GetMeta().GetName(),
			CreationTime:     r.GetMeta().GetCreationTime(),
			ModificationTime: r.GetMeta().GetModificationTime(),
			Labels:           r.GetMeta().GetLabels(),
		}
	}
	return meta
}

func (f *from) ResourceList(rs model.ResourceList) *ResourceList {
	items := make([]Resource, len(rs.GetItems()))
	for i, r := range rs.GetItems() {
		items[i] = f.Resource(r)
	}
	return &ResourceList{
		Total: rs.GetPagination().Total,
		Items: items,
	}
}

var To = &to{}

type to struct{}

func (t *to) Core(r Resource) (model.Resource, error) {
	resource, err := registry.Global().NewObject(model.ResourceType(r.GetMeta().Type))
	if err != nil {
		return nil, err
	}
	resource.SetMeta(r.GetMeta())
	if err := resource.SetSpec(r.GetSpec()); err != nil {
		return nil, err
	}
	return resource, nil
}
