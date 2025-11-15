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

package bootstrap

import (
	"fmt"
	"os"

	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/features"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh/kubemesh"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh/meshwatcher"
	"github.com/apache/dubbo-kubernetes/pkg/filewatcher"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	"github.com/apache/dubbo-kubernetes/pkg/ptr"
	"sigs.k8s.io/yaml"
)

const (
	defaultMeshConfigMapName = "dubbo"
)

func (s *Server) initMeshConfiguration(args *PlanetArgs, fileWatcher filewatcher.FileWatcher) {
	log.Infof("initializing mesh configuration %v", args.MeshConfigFile)
	col := s.getMeshConfiguration(args, fileWatcher)
	col.AsCollection().WaitUntilSynced(s.internalStop)
	s.environment.Watcher = meshwatcher.ConfigAdapter(col)
	log.Infof("mesh configuration: %s", meshwatcher.PrettyFormatOfMeshConfig(s.environment.Mesh()))
	argsdump, _ := yaml.Marshal(args)
	log.Infof("flags: \n%s", argsdump)
}

func (s *Server) getMeshConfiguration(args *PlanetArgs, fileWatcher filewatcher.FileWatcher) krt.Singleton[meshwatcher.MeshConfigResource] {
	opts := krt.NewOptionsBuilder(s.internalStop, "", args.KrtDebugger)
	sources := s.getConfigurationSources(args, fileWatcher, args.MeshConfigFile, kubemesh.MeshConfigKey)
	if len(sources) == 0 {
		fmt.Printf("\nUsing default mesh - missing file %s and no k8s client\n", args.MeshConfigFile)
	}
	return meshwatcher.NewCollection(opts, sources...)
}

func (s *Server) initMeshNetworks(args *PlanetArgs, fileWatcher filewatcher.FileWatcher) {
	log.Infof("initializing mesh networks configuration %v", args.NetworksConfigFile)
	col := s.getMeshNetworks(args, fileWatcher)
	col.AsCollection().WaitUntilSynced(s.internalStop)
	s.environment.NetworksWatcher = meshwatcher.NetworksAdapter(col)
	log.Infof("mesh networks configuration: %s", meshwatcher.PrettyFormatOfMeshNetworks(s.environment.MeshNetworks()))
}

func (s *Server) getMeshNetworks(args *PlanetArgs, fileWatcher filewatcher.FileWatcher) krt.Singleton[meshwatcher.MeshNetworksResource] {
	// We need to get mesh networks up-front, before we start anything, so we use internalStop rather than scheduling a task to run
	// later.
	opts := krt.NewOptionsBuilder(s.internalStop, "", args.KrtDebugger)
	sources := s.getConfigurationSources(args, fileWatcher, args.NetworksConfigFile, kubemesh.MeshNetworksKey)
	if len(sources) == 0 {
		log.Infof("Using default mesh networks - missing file %s and no k8s client", args.NetworksConfigFile)
	}
	return meshwatcher.NewNetworksCollection(opts, sources...)
}

func (s *Server) getConfigurationSources(args *PlanetArgs, fileWatcher filewatcher.FileWatcher, file string, cmKey string) []meshwatcher.MeshConfigSource {
	opts := krt.NewOptionsBuilder(s.internalStop, "", args.KrtDebugger)
	var userMeshConfig *meshwatcher.MeshConfigSource
	if features.SharedMeshConfig != "" && s.kubeClient != nil {
		userMeshConfig = ptr.Of(kubemesh.NewConfigMapSource(s.kubeClient, args.Namespace, features.SharedMeshConfig, cmKey, opts))
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		fileSource, err := meshwatcher.NewFileSource(fileWatcher, file, opts)
		if err == nil {
			return toSources(fileSource, userMeshConfig)
		}
	}

	if s.kubeClient == nil {
		return nil
	}

	configMapName := getMeshConfigMapName("")
	primary := kubemesh.NewConfigMapSource(s.kubeClient, args.Namespace, configMapName, cmKey, opts)
	return toSources(primary, userMeshConfig)
}

func toSources(base meshwatcher.MeshConfigSource, user *meshwatcher.MeshConfigSource) []meshwatcher.MeshConfigSource {
	if user != nil {
		return []meshwatcher.MeshConfigSource{*user, base}
	}
	return []meshwatcher.MeshConfigSource{base}
}

func getMeshConfigMapName(revision string) string {
	name := defaultMeshConfigMapName
	if revision == "" || revision == "default" {
		return name
	}
	return name + "-" + revision
}
