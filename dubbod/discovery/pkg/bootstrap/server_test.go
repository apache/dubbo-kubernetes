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

package bootstrap

import (
	"testing"

	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
)

func TestConfigKindForSchemaIdentifierIncludesDxgateService(t *testing.T) {
	got, found := configKindForSchemaIdentifier("DxgateService")
	if !found {
		t.Fatal("DxgateService schema identifier was not mapped")
	}
	if got != kind.DxgateService {
		t.Fatalf("kind = %v, want %v", got, kind.DxgateService)
	}
}

func TestConfigKindForSchemaIdentifierIncludesTelemetry(t *testing.T) {
	got, found := configKindForSchemaIdentifier("Telemetry")
	if !found {
		t.Fatal("Telemetry schema identifier was not mapped")
	}
	if got != kind.Telemetry {
		t.Fatalf("kind = %v, want %v", got, kind.Telemetry)
	}
}
