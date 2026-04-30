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

package xds

import (
	"testing"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
)

func TestPushQueueEnqueueMergesPendingRequest(t *testing.T) {
	queue := NewPushQueue()
	con := &Connection{}

	queue.Enqueue(con, &model.PushRequest{
		Reason:         model.NewReasonStats(model.EndpointUpdate),
		ConfigsUpdated: sets.New(model.ConfigKey{Kind: kind.Service, Name: "nginx.app.svc.cluster.local", Namespace: "app"}),
	})
	queue.Enqueue(con, &model.PushRequest{
		Reason:         model.NewReasonStats(model.ConfigUpdate),
		ConfigsUpdated: sets.New(model.ConfigKey{Kind: kind.DestinationRule, Name: "nginx", Namespace: "app"}),
		Full:           true,
	})

	stats := queue.Stats()
	if stats.Pending != 1 || stats.Queued != 1 {
		t.Fatalf("Stats = %+v, want one merged pending request", stats)
	}

	_, req, shutdown := queue.Dequeue()
	if shutdown {
		t.Fatalf("Dequeue() shutdown = true, want false")
	}
	if !req.Full {
		t.Fatalf("Full = false, want true")
	}
	if got := req.Reason[model.EndpointUpdate]; got != 1 {
		t.Fatalf("EndpointUpdate reason count = %d, want 1", got)
	}
	if got := req.Reason[model.ConfigUpdate]; got != 1 {
		t.Fatalf("ConfigUpdate reason count = %d, want 1", got)
	}
	if len(req.ConfigsUpdated) != 2 {
		t.Fatalf("ConfigsUpdated = %v, want both config keys", req.ConfigsUpdated)
	}
}
