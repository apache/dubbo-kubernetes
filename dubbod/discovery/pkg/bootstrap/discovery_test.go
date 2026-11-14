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
	"testing"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/xds"
	v1 "github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/xds/v1"
)

func TestInitGeneratorsRegistersTriplePrefix(t *testing.T) {
	server := xds.NewDiscoveryServer(model.NewEnvironment(), nil)
	InitGenerators(server)

	expectedTypes := []string{v1.EndpointType, v1.ListenerType, v1.RouteType, v1.ClusterType}
	if len(server.Generators) != len(expectedTypes) {
		t.Fatalf("generator count = %d, want %d", len(server.Generators), len(expectedTypes))
	}
	for _, typeURL := range expectedTypes {
		if server.Generators["triple/"+typeURL] == nil {
			t.Fatalf("missing generator for triple/%s", typeURL)
		}
	}
}
