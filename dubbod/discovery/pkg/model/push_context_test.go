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

package model

import (
	"testing"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
)

func TestPushRequestCopyMergePreservesQueuedUpdates(t *testing.T) {
	oldPush := NewPushContext()
	newPush := NewPushContext()
	start := time.Unix(100, 0)

	existing := &PushRequest{
		Reason:           NewReasonStats(EndpointUpdate),
		ConfigsUpdated:   sets.New(ConfigKey{Kind: kind.Service, Name: "nginx.app.svc.cluster.local", Namespace: "app"}),
		AddressesUpdated: sets.New("10.0.0.1"),
		Push:             oldPush,
		Start:            start,
		Delta: ResourceDelta{
			Subscribed: sets.New("outbound|80||nginx.app.svc.cluster.local"),
		},
	}
	incoming := &PushRequest{
		Reason:           NewReasonStats(ConfigUpdate),
		ConfigsUpdated:   sets.New(ConfigKey{Kind: kind.MeshService, Name: "nginx", Namespace: "app"}),
		AddressesUpdated: sets.New("10.0.0.2"),
		Full:             true,
		Forced:           true,
		Push:             newPush,
		Start:            start.Add(time.Second),
		Delta: ResourceDelta{
			Unsubscribed: sets.New("outbound|80||old.app.svc.cluster.local"),
		},
	}

	merged := existing.CopyMerge(incoming)
	if merged == existing || merged == incoming {
		t.Fatalf("CopyMerge returned an input request")
	}
	if !merged.Full || !merged.Forced {
		t.Fatalf("Full/Forced = %v/%v, want true/true", merged.Full, merged.Forced)
	}
	if merged.Push != newPush {
		t.Fatalf("Push was not updated to latest context")
	}
	if !merged.Start.Equal(start) {
		t.Fatalf("Start = %v, want %v", merged.Start, start)
	}
	if got := merged.Reason[EndpointUpdate]; got != 1 {
		t.Fatalf("EndpointUpdate reason count = %d, want 1", got)
	}
	if got := merged.Reason[ConfigUpdate]; got != 1 {
		t.Fatalf("ConfigUpdate reason count = %d, want 1", got)
	}
	if len(merged.ConfigsUpdated) != 2 {
		t.Fatalf("ConfigsUpdated = %v, want 2 entries", merged.ConfigsUpdated)
	}
	if !merged.AddressesUpdated.Contains("10.0.0.1") || !merged.AddressesUpdated.Contains("10.0.0.2") {
		t.Fatalf("AddressesUpdated = %v, want both addresses", merged.AddressesUpdated)
	}
	if !merged.Delta.Subscribed.Contains("outbound|80||nginx.app.svc.cluster.local") {
		t.Fatalf("Subscribed delta = %v, want nginx cluster", merged.Delta.Subscribed)
	}
	if !merged.Delta.Unsubscribed.Contains("outbound|80||old.app.svc.cluster.local") {
		t.Fatalf("Unsubscribed delta = %v, want old cluster", merged.Delta.Unsubscribed)
	}

	merged.ConfigsUpdated.Insert(ConfigKey{Kind: kind.Service, Name: "api.app.svc.cluster.local", Namespace: "app"})
	if len(existing.ConfigsUpdated) != 1 {
		t.Fatalf("CopyMerge mutated existing ConfigsUpdated: %v", existing.ConfigsUpdated)
	}
}

func TestPushContextStatusJSONEmptyObject(t *testing.T) {
	data, err := NewPushContext().StatusJSON()
	if err != nil {
		t.Fatalf("StatusJSON() failed: %v", err)
	}
	if string(data) != "{}" {
		t.Fatalf("StatusJSON() = %s, want {}", string(data))
	}
}
