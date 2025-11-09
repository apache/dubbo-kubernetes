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

type MeshConfigSource = krt.Singleton[string]

func NewFileSource(fileWatcher filewatcher.FileWatcher, filename string, opts krt.OptionsBuilder) (MeshConfigSource, error) {
	return krtfiles.NewFileSingleton[string](fileWatcher, filename, func(filename string) (string, error) {
		// #nosec G304 -- filename is a controlled path from configuration
		b, err := os.ReadFile(filename)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}, opts.WithName("Mesh_File_"+path.Base(filename))...)
}

func NewCollection(opts krt.OptionsBuilder, sources ...MeshConfigSource) krt.Singleton[MeshConfigResource] {
	if len(sources) > 2 {
		panic("currently only 2 sources are supported")
	}
	return krt.NewSingleton[MeshConfigResource](
		func(ctx krt.HandlerContext) *MeshConfigResource {
			meshCfg := mesh.DefaultMeshConfig()

			for _, attempt := range sources {
				s := krt.FetchOne(ctx, attempt.AsCollection())
				if s == nil {
					log.Debugf("mesh configuration source missing")
					continue
				}
				n, err := mesh.ApplyMeshConfig(*s, meshCfg)
				if err != nil {
					if len(sources) == 1 {
						log.Errorf("invalid mesh config, using last known state: %v", err)
						ctx.DiscardResult()
						return &MeshConfigResource{mesh.DefaultMeshConfig()}
					}
					log.Errorf("invalid mesh config, ignoring: %v", err)
					continue
				}
				meshCfg = n
			}
			return &MeshConfigResource{meshCfg}
		}, opts.WithName("MeshConfig")...,
	)
}

// NewNetworksCollection builds a new meshnetworks config built by applying the provided sources.
// Sources are applied in order (example: default < sources[0] < sources[1]).
func NewNetworksCollection(opts krt.OptionsBuilder, sources ...MeshConfigSource) krt.Singleton[MeshNetworksResource] {
	if len(sources) > 2 {
		// There is no real reason for this other than to enforce we don't accidentally put more sources
		panic("currently only 2 sources are supported")
	}
	return krt.NewSingleton[MeshNetworksResource](
		func(ctx krt.HandlerContext) *MeshNetworksResource {
			for _, attempt := range sources {
				if s := krt.FetchOne(ctx, attempt.AsCollection()); s != nil {
					n, err := mesh.ParseMeshNetworks(*s)
					if err != nil {
						log.Errorf("invalid mesh networks, using last known state: %v", err)
						ctx.DiscardResult()
						return &MeshNetworksResource{mesh.DefaultMeshNetworks()}
					}
					return &MeshNetworksResource{n}
				}
			}
			return &MeshNetworksResource{nil}
		}, opts.WithName("MeshNetworks")...,
	)
}
