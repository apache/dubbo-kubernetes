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

package traditional

import (
	"fmt"
	"sigs.k8s.io/yaml"
	"strings"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
)

func GenerateCpGroupPath(resourceName string, name string) string {
	return pathSeparator + cpGroup + pathSeparator + resourceName + pathSeparator + name
}

func getMappingPath(keys ...string) string {
	rootDir := pathSeparator + dubboGroup + pathSeparator + mappingGroup + pathSeparator
	for i := 0; i < len(keys); i++ {
		if i == len(keys)-1 {
			// 遍历到了最后一个元素
			rootDir += keys[i]
		} else {
			rootDir += fmt.Sprintf("%s%s", keys[i], pathSeparator)
		}
	}
	return rootDir
}

func getMetadataPath(keys ...string) string {
	if len(keys) == 0 {
		return pathSeparator + dubboGroup + pathSeparator + metadataGroup
	}
	rootDir := pathSeparator + dubboGroup + pathSeparator + metadataGroup + pathSeparator
	for i := 0; i < len(keys); i++ {
		if i == len(keys)-1 {
			// 遍历到了最后一个元素
			rootDir += keys[i]
		} else {
			rootDir += fmt.Sprintf("%s%s", keys[i], pathSeparator)
		}
	}
	return rootDir
}

func getDubboCpPath(keys ...string) string {
	rootDir := pathSeparator + cpGroup + pathSeparator
	for i := 0; i < len(keys); i++ {
		if i == len(keys)-1 {
			// 遍历到了最后一个元素
			rootDir += keys[i]
		} else {
			rootDir += fmt.Sprintf("%s%s", keys[i], pathSeparator)
		}
	}
	return rootDir
}

func splitAppAndRevision(name string) (app string, revision string) {
	split := strings.Split(name, "-")
	n := len(split)
	app = strings.Replace(name, "-"+split[n-1], "", -1)
	return app, split[n-1]
}

func parseTagConfig(rawRouteData string) (*mesh_proto.TagRoute, error) {
	tagRoute := &mesh_proto.TagRoute{}
	err := yaml.Unmarshal([]byte(rawRouteData), tagRoute)
	return tagRoute, err
}

func parseConfiguratorConfig(rawRouteData string) (*mesh_proto.DynamicConfig, error) {
	routerConfig := &mesh_proto.DynamicConfig{}
	err := yaml.Unmarshal([]byte(rawRouteData), routerConfig)
	return routerConfig, err
}

func parseConditionConfig(rawRouteData string) (*mesh_proto.ConditionRoute, error) {
	return mesh_proto.ConditionRouteDecodeFromYAML([]byte(rawRouteData))
}

func parseAffinityConfig(rawRouteData string) (*mesh_proto.AffinityRoute, error) {
	affinityRoute := &mesh_proto.AffinityRoute{}
	err := yaml.Unmarshal([]byte(rawRouteData), affinityRoute)
	return affinityRoute, err
}
