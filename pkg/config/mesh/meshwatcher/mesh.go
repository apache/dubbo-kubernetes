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
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	"github.com/apache/dubbo-kubernetes/pkg/util/protomarshal"
	meshv1alpha1 "github.com/kdubbo/api/mesh/v1alpha1"
	"google.golang.org/protobuf/proto"
)

// MeshGlobalSetupResource holds the current MeshGlobalSetup state
type MeshGlobalSetupResource struct {
	*meshv1alpha1.MeshGlobalSetup
}

type adapter struct {
	krt.Singleton[MeshGlobalSetupResource]
}

var _ mesh.Watcher = adapter{}

type WatcherCollection interface {
	mesh.Watcher
	krt.Singleton[MeshGlobalSetupResource]
}

func (a adapter) Mesh() *meshv1alpha1.MeshGlobalSetup {
	// Just get the value; we know there is always one set due to the way the collection is setup.
	v := a.Singleton.Get()
	return v.MeshGlobalSetup
}

func (a adapter) AddMeshHandler(h func()) *mesh.WatcherHandlerRegistration {
	// Do not run initial state to match existing semantics
	colReg := a.Singleton.AsCollection().RegisterBatch(func(o []krt.Event[MeshGlobalSetupResource]) {
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

func (m MeshGlobalSetupResource) ResourceName() string { return "MeshGlobalSetupResource" }

func (m MeshGlobalSetupResource) Equals(other MeshGlobalSetupResource) bool {
	return proto.Equal(m.MeshGlobalSetup, other.MeshGlobalSetup)
}

func ConfigAdapter(configuration krt.Singleton[MeshGlobalSetupResource]) WatcherCollection {
	return adapter{configuration}
}

func PrettyFormatOfMeshGlobalSetup(meshGlobalSetup *meshv1alpha1.MeshGlobalSetup) string {
	meshGlobalSetupDump, _ := protomarshal.ToYAML(meshGlobalSetup)
	return meshGlobalSetupDump
}
