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

package activation

import (
	"errors"
	"io"
	"strings"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/activation/demandpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DemandService receives the demand gateways broadcast to every control-plane
// replica.
type DemandService struct {
	demandpb.UnimplementedActivationDemandServer

	registry *Registry
}

func NewDemandService(registry *Registry) *DemandService {
	return &DemandService{registry: registry}
}

// Report consumes one gateway's snapshot stream until it ends.
//
// The stream's lifetime is the gateway's liveness. On any exit the gateway is
// forgotten immediately rather than left to age out, so a rolling gateway
// update does not hold a workload scaled up for a full TTL after the old pod
// is gone.
func (s *DemandService) Report(stream demandpb.ActivationDemand_ReportServer) error {
	reporter := ""
	var snapshots int64

	defer func() {
		if reporter != "" {
			s.registry.Forget(reporter)
		}
	}()

	for {
		snapshot, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return stream.SendAndClose(&demandpb.ReportSummary{Snapshots: snapshots})
		}
		if err != nil {
			return err
		}

		name := strings.TrimSpace(snapshot.GetReporter())
		if name == "" {
			return status.Error(codes.InvalidArgument, "demand snapshot is missing a reporter identity")
		}
		// Two identities on one stream would leave the first one's demand
		// behind with nothing refreshing it.
		if reporter != "" && name != reporter {
			return status.Errorf(codes.InvalidArgument,
				"reporter changed mid-stream from %q to %q", reporter, name)
		}
		reporter = name

		pending, err := targetsOf(snapshot)
		if err != nil {
			return err
		}
		s.registry.ReportSnapshot(reporter, pending)
		snapshots++
	}
}

func targetsOf(snapshot *demandpb.DemandSnapshot) (map[Target]int64, error) {
	pending := make(map[Target]int64, len(snapshot.GetTargets()))
	for _, item := range snapshot.GetTargets() {
		namespace := strings.TrimSpace(item.GetNamespace())
		service := strings.TrimSpace(item.GetService())
		if namespace == "" || service == "" {
			return nil, status.Errorf(codes.InvalidArgument,
				"demand target is missing a namespace or service: %q/%q", namespace, service)
		}
		// Summed rather than overwritten: a malformed snapshot that repeats a
		// target must not silently discard one of the counts.
		pending[Target{Namespace: namespace, Name: service}] += item.GetPending()
	}
	return pending, nil
}
