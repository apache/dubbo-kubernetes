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
	"strings"
)

import (
	"gopkg.in/yaml.v2"
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
	routeDecoder := yaml.NewDecoder(strings.NewReader(rawRouteData))
	tagRouterConfig := &mesh_proto.TagRoute{}
	err := routeDecoder.Decode(tagRouterConfig)
	if err != nil {
		return nil, err
	}
	return tagRouterConfig, nil
}

func parseConfiguratorConfig(rawRouteData string) (*mesh_proto.DynamicConfig, error) {
	routeDecoder := yaml.NewDecoder(strings.NewReader(rawRouteData))
	routerConfig := &mesh_proto.DynamicConfig{}
	err := routeDecoder.Decode(routerConfig)
	if err != nil {
		return nil, err
	}
	return routerConfig, nil
}

func parseConditionConfig(rawRouteData string) (*mesh_proto.ConditionRoute, error) {
	if v3 := new(mesh_proto.ConditionRouteV3); yaml.Unmarshal([]byte(rawRouteData), &v3) == nil {
		return v3.ToConditionRoute(), nil
	} else if v3x1 := new(mesh_proto.ConditionRouteV3X1); yaml.Unmarshal([]byte(rawRouteData), &v3x1) == nil {
		return v3x1.ToConditionRoute(), nil
	} else {
		return nil, fmt.Errorf("failed to parse condition route")
	}
}
