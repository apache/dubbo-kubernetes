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
		ConfigsUpdated: sets.New(model.ConfigKey{Kind: kind.PeerAuthentication, Name: "default", Namespace: "app"}),
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

func TestPushQueueRequeuesUpdatesArrivingDuringProcessing(t *testing.T) {
	queue := NewPushQueue()
	con := &Connection{}
	initial := &model.PushRequest{
		Reason: model.NewReasonStats(model.EndpointUpdate),
	}
	queue.Enqueue(con, initial)

	gotCon, gotReq, shutdown := queue.Dequeue()
	if shutdown || gotCon != con || gotReq != initial {
		t.Fatalf("first Dequeue() = %p, %p, %v; want %p, %p, false", gotCon, gotReq, shutdown, con, initial)
	}
	queue.Enqueue(con, &model.PushRequest{
		Full:   true,
		Reason: model.NewReasonStats(model.ConfigUpdate),
	})
	if stats := queue.Stats(); stats.Processing != 1 || stats.Pending != 0 || stats.Queued != 0 {
		t.Fatalf("Stats while processing = %+v, want processing=1 only", stats)
	}

	queue.MarkDone(con)
	if stats := queue.Stats(); stats.Processing != 0 || stats.Pending != 1 || stats.Queued != 1 {
		t.Fatalf("Stats after MarkDone = %+v, want one requeued request", stats)
	}
	_, merged, shutdown := queue.Dequeue()
	if shutdown || !merged.Full || merged.Reason[model.ConfigUpdate] != 1 {
		t.Fatalf("second Dequeue() = %+v, shutdown=%v; want full config update", merged, shutdown)
	}
}

func TestPushQueueShutdownUnblocksDequeueAndRejectsEnqueue(t *testing.T) {
	queue := NewPushQueue()
	result := make(chan bool, 1)
	go func() {
		_, _, shutdown := queue.Dequeue()
		result <- shutdown
	}()

	queue.ShutDown()
	if shutdown := <-result; !shutdown {
		t.Fatal("Dequeue() shutdown = false, want true")
	}
	queue.Enqueue(&Connection{}, &model.PushRequest{})
	if stats := queue.Stats(); stats != (pushQueueStats{}) {
		t.Fatalf("Stats after shutdown enqueue = %+v, want empty", stats)
	}
}
