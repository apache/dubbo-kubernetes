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

package render

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestGenerateManifestUsesEmbeddedInstallerAssets(t *testing.T) {
	manifests, _, err := GenerateManifest(nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("GenerateManifest() error = %v", err)
	}
	if len(manifests) == 0 {
		t.Fatal("GenerateManifest() returned zero manifest sets")
	}
}

func TestGenerateManifestExposesDubbodPrometheusScrapeEndpoint(t *testing.T) {
	manifests, _, err := GenerateManifest(nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("GenerateManifest() error = %v", err)
	}

	for _, set := range manifests {
		for _, m := range set.Manifests {
			if m.GetKind() != "Service" || m.GetName() != "dubbod" || m.GetNamespace() != "dubbo-system" {
				continue
			}
			annotations := m.GetAnnotations()
			if annotations["prometheus.io/scrape"] != "true" {
				t.Fatalf("prometheus.io/scrape = %q, want true", annotations["prometheus.io/scrape"])
			}
			if annotations["prometheus.io/path"] != "/metrics" {
				t.Fatalf("prometheus.io/path = %q, want /metrics", annotations["prometheus.io/path"])
			}
			if annotations["prometheus.io/port"] != "8080" {
				t.Fatalf("prometheus.io/port = %q, want 8080", annotations["prometheus.io/port"])
			}

			ports, ok, err := unstructured.NestedSlice(m.Object, "spec", "ports")
			if err != nil || !ok {
				t.Fatalf("spec.ports missing: ok=%v err=%v", ok, err)
			}
			for _, port := range ports {
				p, ok := port.(map[string]interface{})
				if !ok {
					continue
				}
				name, _, _ := unstructured.NestedString(p, "name")
				number, _, _ := unstructured.NestedInt64(p, "port")
				if name == "http-monitoring" && number == 8080 {
					return
				}
			}
			t.Fatal("dubbod Service missing http-monitoring port 8080")
		}
	}
	t.Fatal("dubbod Service not rendered")
}
