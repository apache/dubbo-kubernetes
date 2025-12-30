//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package meshwatcher

import (
	meshv1alpha1 "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	"github.com/apache/dubbo-kubernetes/pkg/util/protomarshal"
	"google.golang.org/protobuf/proto"
)

// MeshGlobalConfigResource holds the current MeshGlobalConfig state
type MeshGlobalConfigResource struct {
	*meshv1alpha1.MeshGlobalConfig
}

type adapter struct {
	krt.Singleton[MeshGlobalConfigResource]
}

var _ mesh.Watcher = adapter{}

type WatcherCollection interface {
	mesh.Watcher
	krt.Singleton[MeshGlobalConfigResource]
}

func (a adapter) Mesh() *meshv1alpha1.MeshGlobalConfig {
	// Just get the value; we know there is always one set due to the way the collection is setup.
	v := a.Singleton.Get()
	return v.MeshGlobalConfig
}

func (a adapter) AddMeshHandler(h func()) *mesh.WatcherHandlerRegistration {
	// Do not run initial state to match existing semantics
	colReg := a.Singleton.AsCollection().RegisterBatch(func(o []krt.Event[MeshGlobalConfigResource]) {
		h()
	}, false)
	reg := mesh.NewWatcherHandlerRegistration(func() {
		colReg.UnregisterHandler()
	})
	return reg
}

// DeleteMeshHandler removes a previously registered handler.
func (a adapter) DeleteMeshHandler(registration *mesh.WatcherHandlerRegistration) {
	registration.Remove()
}

func (m MeshGlobalConfigResource) ResourceName() string { return "MeshGlobalConfigResource" }

func (m MeshGlobalConfigResource) Equals(other MeshGlobalConfigResource) bool {
	return proto.Equal(m.MeshGlobalConfig, other.MeshGlobalConfig)
}

func ConfigAdapter(configuration krt.Singleton[MeshGlobalConfigResource]) WatcherCollection {
	return adapter{configuration}
}

func PrettyFormatOfMeshGlobalConfig(meshGlobalConfig *meshv1alpha1.MeshGlobalConfig) string {
	meshGlobalConfigDump, _ := protomarshal.ToYAML(meshGlobalConfig)
	return meshGlobalConfigDump
}
