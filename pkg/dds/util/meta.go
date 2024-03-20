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

package util

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"golang.org/x/exp/maps"
	"time"
)

// DDS ResourceMeta only contains name and mesh.
// The rest is managed by the receiver of resources anyways. See ResourceSyncer#Sync
type resourceMeta struct {
	name   string
	mesh   string
	labels map[string]string
}

type CloneResourceMetaOpt func(*resourceMeta)

func WithName(name string) CloneResourceMetaOpt {
	return func(m *resourceMeta) {
		if m.labels[mesh_proto.DisplayName] == "" {
			m.labels[mesh_proto.DisplayName] = m.name
		}
		m.name = name
	}
}

func WithLabel(key, value string) CloneResourceMetaOpt {
	return func(m *resourceMeta) {
		m.labels[key] = value
	}
}

func CloneResourceMeta(m model.ResourceMeta, fs ...CloneResourceMetaOpt) model.ResourceMeta {
	labels := maps.Clone(m.GetLabels())
	if labels == nil {
		labels = map[string]string{}
	}
	meta := &resourceMeta{
		name:   m.GetName(),
		mesh:   m.GetMesh(),
		labels: labels,
	}
	for _, f := range fs {
		f(meta)
	}
	if len(meta.labels) == 0 {
		meta.labels = nil
	}
	return meta
}

func DubboResourceMetaToResourceMeta(meta *mesh_proto.DubboResource_Meta) model.ResourceMeta {
	return &resourceMeta{
		name:   meta.Name,
		mesh:   meta.Mesh,
		labels: meta.GetLabels(),
	}
}

func (r *resourceMeta) GetName() string {
	return r.name
}

func (r *resourceMeta) GetNameExtensions() model.ResourceNameExtensions {
	return model.ResourceNameExtensionsUnsupported
}

func (r *resourceMeta) GetVersion() string {
	return ""
}

func (r *resourceMeta) GetMesh() string {
	return r.mesh
}

func (r *resourceMeta) GetCreationTime() time.Time {
	return time.Unix(0, 0)
}

func (r *resourceMeta) GetModificationTime() time.Time {
	return time.Unix(0, 0)
}

func (r *resourceMeta) GetLabels() map[string]string {
	return r.labels
}
