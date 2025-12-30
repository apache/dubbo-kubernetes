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

package bootstrap

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/util/ptr"
	"os"

	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/features"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh/kubemesh"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh/meshwatcher"
	"github.com/apache/dubbo-kubernetes/pkg/filewatcher"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	"sigs.k8s.io/yaml"
)

const (
	defaultMeshGlobalConfigMapName = "dubbo"
)

func (s *Server) initMeshGlobalConfiguration(args *PlanetArgs, fileWatcher filewatcher.FileWatcher) {
	log.Infof("initializing mesh global configuration %v", args.MeshGlobalConfigFile)
	col := s.getMeshGlobalConfiguration(args, fileWatcher)
	col.AsCollection().WaitUntilSynced(s.internalStop)
	s.environment.Watcher = meshwatcher.ConfigAdapter(col)
	log.Infof("mesh global configuration: %s", meshwatcher.PrettyFormatOfMeshGlobalConfig(s.environment.Mesh()))
	argsdump, _ := yaml.Marshal(args)
	log.Infof("flags: \n%s", argsdump)
}

func (s *Server) getMeshGlobalConfiguration(args *PlanetArgs, fileWatcher filewatcher.FileWatcher) krt.Singleton[meshwatcher.MeshGlobalConfigResource] {
	opts := krt.NewOptionsBuilder(s.internalStop, "", args.KrtDebugger)
	sources := s.getConfigurationSources(args, fileWatcher, args.MeshGlobalConfigFile, kubemesh.MeshGlobalConfigKey)
	if len(sources) == 0 {
		fmt.Printf("\nUsing default mesh - missing file %s and no k8s client\n", args.MeshGlobalConfigFile)
	}
	return meshwatcher.NewCollection(opts, sources...)
}

func (s *Server) getConfigurationSources(args *PlanetArgs, fileWatcher filewatcher.FileWatcher, file string, cmKey string) []meshwatcher.MeshGlobalConfigSource {
	opts := krt.NewOptionsBuilder(s.internalStop, "", args.KrtDebugger)
	var userMeshGlobalConfig *meshwatcher.MeshGlobalConfigSource
	if features.SharedMeshGlobalConfig != "" && s.kubeClient != nil {
		userMeshGlobalConfig = ptr.Of(kubemesh.NewConfigMapSource(s.kubeClient, args.Namespace, features.SharedMeshGlobalConfig, cmKey, opts))
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		fileSource, err := meshwatcher.NewFileSource(fileWatcher, file, opts)
		if err == nil {
			return toSources(fileSource, userMeshGlobalConfig)
		}
	}

	if s.kubeClient == nil {
		return nil
	}

	configMapName := getMeshGlobalConfigMapName("")
	primary := kubemesh.NewConfigMapSource(s.kubeClient, args.Namespace, configMapName, cmKey, opts)
	return toSources(primary, userMeshGlobalConfig)
}

func toSources(base meshwatcher.MeshGlobalConfigSource, user *meshwatcher.MeshGlobalConfigSource) []meshwatcher.MeshGlobalConfigSource {
	if user != nil {
		return []meshwatcher.MeshGlobalConfigSource{*user, base}
	}
	return []meshwatcher.MeshGlobalConfigSource{base}
}

func getMeshGlobalConfigMapName(revision string) string {
	name := defaultMeshGlobalConfigMapName
	if revision == "" || revision == "default" {
		return name
	}
	return name + "-" + revision
}
