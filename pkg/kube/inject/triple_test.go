// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package inject

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	meshv1alpha1 "github.com/dubml/api/mesh/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTripleTemplateRetainsApplicationAndDoesNotAppendGRPC(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	data, err := os.ReadFile(filepath.Join(filepath.Dir(file), "../../../manifests/charts/dubbod/files/triple-engine.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	templates, err := ParseTemplates(RawTemplates{InherentTripleTemplateName: string(data)})
	if err != nil {
		t.Fatal(err)
	}
	values, err := NewValuesConfig("{}")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"app", "business"} {
		for _, image := range []string{"business:v1", "dubml/dubbod:debug"} {
			for _, explicit := range []bool{true, false} {
				t.Run(fmt.Sprintf("%s/%s/explicit=%t", name, image, explicit), func(t *testing.T) {
					original := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "triple-app", Namespace: "app", Annotations: map[string]string{InherentInjectTemplatesAnnoName: InherentTripleTemplateName}}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: name, Image: "business:v1", Command: []string{"/business"}, Args: []string{"serve"}}}}}
					original.Spec.Containers[0].Image = image
					if !explicit {
						original.Spec.Containers[0].Command, original.Spec.Containers[0].Args = nil, nil
					}
					wantCommand, wantArgs := original.Spec.Containers[0].Command, original.Spec.Containers[0].Args
					req := InjectionParameters{
						pod: original, templates: templates, valuesConfig: values,
						meshConfig:  &meshv1alpha1.MeshConfig{TrustDomain: "cluster.local", DefaultConfig: &meshv1alpha1.ProxyConfig{DiscoveryAddress: "dubbod.dubbo-system.svc:26012"}},
						proxyConfig: &meshv1alpha1.ProxyConfig{DiscoveryAddress: "dubbod.dubbo-system.svc:26012"},
					}
					merged, injected, err := RunTemplate(req)
					if err != nil {
						t.Fatal(err)
					}
					if err := postProcessPod(merged, *injected, req); err != nil {
						t.Fatal(err)
					}
					if len(merged.Spec.Containers) != 1 {
						t.Fatalf("unexpected added container: %v", merged.Spec.Containers)
					}
					app := merged.Spec.Containers[0]
					if !reflect.DeepEqual(app.Command, wantCommand) || !reflect.DeepEqual(app.Args, wantArgs) {
						t.Fatalf("startup = %v %v, want %v %v", app.Command, app.Args, wantCommand, wantArgs)
					}
					if merged.Annotations[InherentInjectTemplatesAnnoName] != InherentTripleTemplateName {
						t.Fatal("grpc-engine appended")
					}
					assertDirectXDSConnection(t, merged, name, InherentGRPCSecretNameForMeta(original.ObjectMeta))
					before := merged.DeepCopy()
					if err := postProcessPod(merged, *injected, req); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(before, merged) {
						t.Fatal("repeated injection changed Pod")
					}
				})
			}
		}
	}
}
