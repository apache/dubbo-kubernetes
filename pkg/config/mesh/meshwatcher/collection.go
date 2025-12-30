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
	"os"
	"path"

	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/filewatcher"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	krtfiles "github.com/apache/dubbo-kubernetes/pkg/kube/krt/files"

	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"
)

var log = dubbolog.RegisterScope("meshwatcher", "mesh watcher debugging")

type MeshGlobalConfigSource = krt.Singleton[string]

func NewFileSource(fileWatcher filewatcher.FileWatcher, filename string, opts krt.OptionsBuilder) (MeshGlobalConfigSource, error) {
	return krtfiles.NewFileSingleton[string](fileWatcher, filename, func(filename string) (string, error) {
		b, err := os.ReadFile(filename)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}, opts.WithName("Mesh_File_"+path.Base(filename))...)
}

func NewCollection(opts krt.OptionsBuilder, sources ...MeshGlobalConfigSource) krt.Singleton[MeshGlobalConfigResource] {
	if len(sources) > 2 {
		panic("currently only 2 sources are supported")
	}
	return krt.NewSingleton[MeshGlobalConfigResource](
		func(ctx krt.HandlerContext) *MeshGlobalConfigResource {
			meshCfg := mesh.DefaultMeshGlobalConfig()

			for _, attempt := range sources {
				s := krt.FetchOne(ctx, attempt.AsCollection())
				if s == nil {
					log.Debugf("mesh configuration source missing")
					continue
				}
				n, err := mesh.ApplyMeshGlobalConfig(*s, meshCfg)
				if err != nil {
					if len(sources) == 1 {
						log.Errorf("invalid mesh global config, using last known state: %v", err)
						ctx.DiscardResult()
						return &MeshGlobalConfigResource{mesh.DefaultMeshGlobalConfig()}
					}
					log.Errorf("invalid mesh global config, ignoring: %v", err)
					continue
				}
				meshCfg = n
			}
			return &MeshGlobalConfigResource{meshCfg}
		}, opts.WithName("MeshGlobalConfig")...,
	)
}
