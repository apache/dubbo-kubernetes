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

// Package activation serves KEDA's external scaler contract for Services that
// are scaled to zero.
//
// KEDA owns the replica count. This package only answers "is anything waiting
// on this Service, and how much", so KEDA can take a Service from zero to one
// when a request arrives for it. Nothing here writes replicas, which is what
// keeps a Service from being driven by two controllers at once.
package activation

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/activation/externalscaler"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// serviceMetadataKey names the Service a ScaledObject trigger activates.
	// It is required: guessing it from the ScaledObject name would silently
	// activate the wrong workload when the two do not match.
	serviceMetadataKey = "service"

	// namespaceMetadataKey overrides the namespace, for the rare ScaledObject
	// that lives apart from the Service it scales. Defaults to the
	// ScaledObject's own namespace.
	namespaceMetadataKey = "namespace"

	// targetPendingMetadataKey is the pending-request count HPA aims to keep
	// per replica once the workload is past zero.
	targetPendingMetadataKey = "targetPendingRequests"

	defaultTargetPendingRequests = 1.0

	// metricPrefix keeps this scaler's metric distinguishable from the other
	// triggers on the same ScaledObject.
	metricPrefix = "dubbo-activation"
)

// Scaler implements KEDA's ExternalScaler over a DemandSource.
type Scaler struct {
	externalscaler.UnimplementedExternalScalerServer

	demand DemandSource

	// streams counts the open StreamIsActive calls per target. A target with
	// none is one KEDA is not listening to, which is the difference between a
	// policy that will activate and one that only looks like it will.
	streamsMu sync.Mutex
	streams   map[Target]int
}

func NewScaler(demand DemandSource) *Scaler {
	return &Scaler{
		demand:  demand,
		streams: map[Target]int{},
	}
}

// Subscribed reports whether KEDA currently holds an activation stream for the
// target. The policy controller surfaces this as a status condition, because
// otherwise a misconfigured ScaledObject looks identical to a working one until
// the first request is dropped.
func (s *Scaler) Subscribed(target Target) bool {
	s.streamsMu.Lock()
	defer s.streamsMu.Unlock()
	return s.streams[target] > 0
}

func (s *Scaler) streamOpened(target Target) {
	s.streamsMu.Lock()
	defer s.streamsMu.Unlock()
	s.streams[target]++
}

func (s *Scaler) streamClosed(target Target) {
	s.streamsMu.Lock()
	defer s.streamsMu.Unlock()
	if s.streams[target] <= 1 {
		delete(s.streams, target)
		return
	}
	s.streams[target]--
}

// trigger is one ScaledObject's external trigger, resolved to what this scaler
// needs to answer for it.
type trigger struct {
	target        Target
	metricName    string
	targetPending float64
}

func (s *Scaler) IsActive(_ context.Context, ref *externalscaler.ScaledObjectRef) (*externalscaler.IsActiveResponse, error) {
	parsed, err := parseTrigger(ref)
	if err != nil {
		return nil, err
	}
	return &externalscaler.IsActiveResponse{
		Result: s.demand.Pending(parsed.target) > 0,
	}, nil
}

// StreamIsActive pushes activation as it happens, so a request held at the
// gateway does not wait out KEDA's polling interval before the workload is
// even asked to start.
func (s *Scaler) StreamIsActive(ref *externalscaler.ScaledObjectRef, stream externalscaler.ExternalScaler_StreamIsActiveServer) error {
	parsed, err := parseTrigger(ref)
	if err != nil {
		return err
	}

	s.streamOpened(parsed.target)
	defer s.streamClosed(parsed.target)

	// Subscribe before the first read, or demand arriving in between would be
	// reported by neither the initial send nor an update.
	updates, cancel := s.demand.Subscribe(parsed.target)
	defer cancel()

	// KEDA expects the current state as soon as the stream opens, not only on
	// the next change.
	if err := stream.Send(&externalscaler.IsActiveResponse{
		Result: s.demand.Pending(parsed.target) > 0,
	}); err != nil {
		return err
	}

	ctx := stream.Context()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case pending, ok := <-updates:
			if !ok {
				return nil
			}
			if err := stream.Send(&externalscaler.IsActiveResponse{Result: pending > 0}); err != nil {
				return err
			}
		}
	}
}

func (s *Scaler) GetMetricSpec(_ context.Context, ref *externalscaler.ScaledObjectRef) (*externalscaler.GetMetricSpecResponse, error) {
	parsed, err := parseTrigger(ref)
	if err != nil {
		return nil, err
	}
	return &externalscaler.GetMetricSpecResponse{
		MetricSpecs: []*externalscaler.MetricSpec{{
			MetricName:      parsed.metricName,
			TargetSizeFloat: parsed.targetPending,
			// TargetSize is deprecated upstream but still read by older KEDA
			// releases, so both are set and kept consistent.
			TargetSize: int64(parsed.targetPending),
		}},
	}, nil
}

func (s *Scaler) GetMetrics(_ context.Context, request *externalscaler.GetMetricsRequest) (*externalscaler.GetMetricsResponse, error) {
	parsed, err := parseTrigger(request.GetScaledObjectRef())
	if err != nil {
		return nil, err
	}
	// KEDA echoes back the name from GetMetricSpec. A mismatch means the two
	// calls disagree about what is being measured, which would otherwise show
	// up as a workload that scales on the wrong signal.
	if name := request.GetMetricName(); name != parsed.metricName {
		return nil, status.Errorf(codes.InvalidArgument,
			"unknown metric %q for %s/%s, expected %q",
			name, parsed.target.Namespace, parsed.target.Name, parsed.metricName)
	}

	pending := float64(s.demand.Pending(parsed.target))
	return &externalscaler.GetMetricsResponse{
		MetricValues: []*externalscaler.MetricValue{{
			MetricName:       parsed.metricName,
			MetricValueFloat: pending,
			MetricValue:      int64(pending),
		}},
	}, nil
}

func parseTrigger(ref *externalscaler.ScaledObjectRef) (trigger, error) {
	if ref == nil {
		return trigger{}, status.Error(codes.InvalidArgument, "missing scaled object reference")
	}
	metadata := ref.GetScalerMetadata()

	service := strings.TrimSpace(metadata[serviceMetadataKey])
	if service == "" {
		return trigger{}, status.Errorf(codes.InvalidArgument,
			"scaled object %s/%s: trigger metadata %q is required",
			ref.GetNamespace(), ref.GetName(), serviceMetadataKey)
	}

	namespace := strings.TrimSpace(metadata[namespaceMetadataKey])
	if namespace == "" {
		namespace = strings.TrimSpace(ref.GetNamespace())
	}
	if namespace == "" {
		return trigger{}, status.Errorf(codes.InvalidArgument,
			"scaled object %s: namespace is unknown, set trigger metadata %q",
			ref.GetName(), namespaceMetadataKey)
	}

	targetPending, err := parseTargetPending(metadata[targetPendingMetadataKey])
	if err != nil {
		return trigger{}, status.Errorf(codes.InvalidArgument,
			"scaled object %s/%s: %v", ref.GetNamespace(), ref.GetName(), err)
	}

	target := Target{Namespace: namespace, Name: service}
	return trigger{
		target:        target,
		metricName:    metricName(target),
		targetPending: targetPending,
	}, nil
}

func parseTargetPending(raw string) (float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultTargetPendingRequests, nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("trigger metadata %q is not a number: %q", targetPendingMetadataKey, raw)
	}
	// Zero or negative would make HPA divide by it and demand an unbounded
	// replica count from a single held request.
	if value <= 0 {
		return 0, fmt.Errorf("trigger metadata %q must be greater than zero, got %q", targetPendingMetadataKey, raw)
	}
	return value, nil
}

// metricName is derived from the target rather than the ScaledObject so two
// triggers pointing at the same Service report the same series.
func metricName(target Target) string {
	return fmt.Sprintf("%s-%s-%s", metricPrefix, target.Namespace, target.Name)
}
