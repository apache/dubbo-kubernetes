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

package render

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/operator/pkg/component"
	"github.com/apache/dubbo-kubernetes/operator/pkg/manifest"
	"github.com/apache/dubbo-kubernetes/operator/pkg/values"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/yaml"
)

type patchContext struct {
	Patch       string
	PostProcess func([]byte) ([]byte, error)
}

func postProcess(_ component.Component, manifests []manifest.Manifest, _ values.Map) ([]manifest.Manifest, error) {
	needPatching := map[int][]patchContext{}
	for idx, patches := range needPatching {
		m := manifests[idx]
		baseJSON, err := yaml.YAMLToJSON([]byte(m.Content))
		if err != nil {
			return nil, err
		}
		typed, err := scheme.Scheme.New(m.GroupVersionKind())
		if err != nil {
			return nil, err
		}

		for _, patch := range patches {
			newBytes, err := strategicpatch.StrategicMergePatch(baseJSON, []byte(patch.Patch), typed)
			if err != nil {
				return nil, fmt.Errorf("patch: %v", err)
			}
			if patch.PostProcess != nil {
				newBytes, err = patch.PostProcess(newBytes)
				if err != nil {
					return nil, fmt.Errorf("patch post process: %v", err)
				}
			}
			baseJSON = newBytes
		}
		nm, err := manifest.FromJSON(baseJSON)
		if err != nil {
			return nil, err
		}
		manifests[idx] = nm
	}

	return manifests, nil
}
