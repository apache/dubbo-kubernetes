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
)

func GenerateCpGroupPath(resourceName string, name string) string {
	return pathSeparator + cpGroup + pathSeparator + resourceName + pathSeparator + name
}

func getMappingPath(keys ...string) string {
	rootDir := pathSeparator + dubboGroup + pathSeparator + mappingGroup + pathSeparator
	for _, key := range keys {
		rootDir += fmt.Sprintf("%s%s", key, pathSeparator)
	}
	return rootDir
}

func getMetadataPath(keys ...string) string {
	rootDir := pathSeparator + dubboGroup + pathSeparator + metadataGroup + pathSeparator
	for _, key := range keys {
		rootDir += fmt.Sprintf("%s%s", key, pathSeparator)
	}
	return rootDir
}

func getDubboCpPath(keys ...string) string {
	rootDir := pathSeparator + cpGroup + pathSeparator
	for _, key := range keys {
		rootDir += fmt.Sprintf("%s%s", key, pathSeparator)
	}
	return rootDir
}
