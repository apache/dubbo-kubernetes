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
	"encoding/json"
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

// postProcess applies any manifest manipulation to be done after Helm chart rendering.
func postProcess(_ component.Component, manifests []manifest.Manifest, _ values.Map) ([]manifest.Manifest, error) {
	// needPatching builds a map of manifest index -> patch. This ensures we only do the full round-tripping once per object.
	needPatching := map[int][]patchContext{}

	// Remove namespace field from cluster-scoped resources
	clusterScopedKinds := map[string]bool{
		"MutatingWebhookConfiguration":   true,
		"ValidatingWebhookConfiguration": true,
		"ClusterRole":                    true,
		"ClusterRoleBinding":             true,
		"CustomResourceDefinition":       true,
	}

	for idx := range manifests {
		m := manifests[idx]
		kind := m.GroupVersionKind().Kind
		if clusterScopedKinds[kind] && m.GetNamespace() != "" {
			// Remove namespace field from cluster-scoped resources
			baseJSON, err := yaml.YAMLToJSON([]byte(m.Content))
			if err != nil {
				return nil, err
			}
			// Parse JSON to remove namespace field
			var obj map[string]interface{}
			if err := json.Unmarshal(baseJSON, &obj); err != nil {
				return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
			}
			if metadata, ok := obj["metadata"].(map[string]interface{}); ok {
				delete(metadata, "namespace")
			}
			// Convert back to JSON
			newJSON, err := json.Marshal(obj)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal JSON: %v", err)
			}
			nm, err := manifest.FromJSON(newJSON)
			if err != nil {
				return nil, err
			}
			manifests[idx] = nm
		}
	}

	// For anything needing a patch, apply them.
	for idx, patches := range needPatching {
		m := manifests[idx]
		// Convert to JSON, which the StrategicMergePatch requires
		baseJSON, err := yaml.YAMLToJSON([]byte(m.Content))
		if err != nil {
			return nil, err
		}
		typed, err := scheme.Scheme.New(m.GroupVersionKind())
		if err != nil {
			return nil, err
		}
		// Apply all the patches.
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
		// Rebuild our manifest.
		nm, err := manifest.FromJSON(baseJSON)
		if err != nil {
			return nil, err
		}
		// Update the manifests list.
		manifests[idx] = nm
	}

	return manifests, nil
}
