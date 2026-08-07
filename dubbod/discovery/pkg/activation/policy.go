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
	"fmt"
	"strings"

	metav1alpha1 "github.com/kdubbo/api/meta/v1alpha1"
	networking "github.com/kdubbo/api/networking/v1alpha3"
	clientnetworking "github.com/kdubbo/client-go/pkg/apis/networking/v1alpha3"
)

// Condition types reported on a ServiceActivationPolicy. Each answers a
// different question, and all four have to hold before a request will actually
// be held and replayed. Collapsing them into one would leave an operator
// guessing which half of the path is broken.
const (
	// ConditionAccepted covers the policy itself: well formed, and pointing at
	// objects that exist.
	ConditionAccepted = "Accepted"

	// ConditionEligible covers the target: whether its protocols can be held
	// and replayed at all.
	ConditionEligible = "Eligible"

	// ConditionScalerReady covers KEDA: whether it is listening for activation
	// on this target.
	ConditionScalerReady = "ScalerReady"

	// ConditionActivatorReady covers the gateways: whether any of them is in a
	// position to catch a request for this target.
	ConditionActivatorReady = "ActivatorReady"
)

const (
	conditionTrue  = "True"
	conditionFalse = "False"
)

// ServiceLookup reports whether the target Service exists. It is an interface
// so policy evaluation can be tested without a cluster.
type ServiceLookup interface {
	HasService(namespace, name string) bool
}

// StreamLookup reports whether KEDA holds an activation stream for a target.
type StreamLookup interface {
	Subscribed(Target) bool
}

// ReporterLookup reports how many gateways stand ready to hold requests for a
// target.
type ReporterLookup interface {
	Reporters(Target) int
}

// PolicyEvaluator turns a policy plus live state into the conditions published
// on its status.
type PolicyEvaluator struct {
	Services  ServiceLookup
	Streams   StreamLookup
	Reporters ReporterLookup
}

// Evaluate returns the conditions for one policy, in a stable order so an
// unchanged policy does not produce a status update on every resync.
func (e PolicyEvaluator) Evaluate(policy *clientnetworking.ServiceActivationPolicy) []*metav1alpha1.DubboCondition {
	generation := policy.GetGeneration()
	spec := &policy.Spec

	accepted, reason := e.accepted(policy.GetNamespace(), spec)
	conditions := []*metav1alpha1.DubboCondition{
		condition(ConditionAccepted, accepted, reason, generation),
	}

	// The remaining conditions describe a path that only exists once the policy
	// is accepted. Reporting them against an unresolved target would invent
	// answers about a Service that may not be the intended one.
	if !accepted {
		conditions = append(conditions,
			condition(ConditionEligible, false, "PolicyNotAccepted", generation),
			condition(ConditionScalerReady, false, "PolicyNotAccepted", generation),
			condition(ConditionActivatorReady, false, "PolicyNotAccepted", generation),
		)
		return conditions
	}

	target := targetOf(policy)
	eligible, eligibleReason := eligible(spec)
	conditions = append(conditions, condition(ConditionEligible, eligible, eligibleReason, generation))

	scalerReady := e.Streams != nil && e.Streams.Subscribed(target)
	conditions = append(conditions,
		condition(ConditionScalerReady, scalerReady, scalerReason(scalerReady), generation))

	activatorReady := e.Reporters != nil && e.Reporters.Reporters(target) > 0
	conditions = append(conditions,
		condition(ConditionActivatorReady, activatorReady, activatorReason(activatorReady), generation))

	return conditions
}

func (e PolicyEvaluator) accepted(namespace string, spec *networking.ServiceActivationPolicy) (bool, string) {
	target := spec.GetTargetRef()
	if target == nil || strings.TrimSpace(target.GetName()) == "" {
		return false, "TargetRefMissing"
	}
	// Only Services have endpoints to wait on; anything else would leave the
	// gateway holding requests for something that never becomes routable.
	if kind := target.GetKind(); kind != "" && kind != "Service" {
		return false, "TargetKindUnsupported"
	}
	if group := target.GetGroup(); group != "" {
		return false, "TargetGroupUnsupported"
	}
	if autoscaler := spec.GetAutoscalerRef(); autoscaler == nil || strings.TrimSpace(autoscaler.GetName()) == "" {
		return false, "AutoscalerRefMissing"
	}
	if e.Services != nil && !e.Services.HasService(namespace, target.GetName()) {
		return false, "TargetServiceNotFound"
	}
	return true, "Accepted"
}

// eligible rejects protocols that cannot survive being held. A stream cannot be
// replayed once the backend is up, so holding one only delays the failure.
func eligible(spec *networking.ServiceActivationPolicy) (bool, string) {
	for _, protocol := range spec.GetProtocols() {
		switch protocol {
		case networking.ActivationProtocol_ACTIVATION_PROTOCOL_UNSPECIFIED,
			networking.ActivationProtocol_HTTP,
			networking.ActivationProtocol_GRPC_UNARY,
			networking.ActivationProtocol_TRIPLE_UNARY:
		default:
			return false, "ProtocolNotActivatable"
		}
	}
	return true, "Eligible"
}

func scalerReason(ready bool) string {
	if ready {
		return "ScalerSubscribed"
	}
	// The usual cause is a ScaledObject that does not point its external
	// trigger at this scaler, or points it at a different Service.
	return "ScalerNotSubscribed"
}

func activatorReason(ready bool) string {
	if ready {
		return "GatewayReporting"
	}
	return "NoGatewayReporting"
}

// targetOf resolves the Service a policy activates. The namespace comes from
// the policy, so a policy can never reach across namespaces into a Service it
// does not own.
func targetOf(policy *clientnetworking.ServiceActivationPolicy) Target {
	return Target{
		Namespace: policy.GetNamespace(),
		Name:      policy.Spec.GetTargetRef().GetName(),
	}
}

func condition(conditionType string, ok bool, reason string, generation int64) *metav1alpha1.DubboCondition {
	value := conditionFalse
	if ok {
		value = conditionTrue
	}
	return &metav1alpha1.DubboCondition{
		Type:               conditionType,
		Status:             value,
		Reason:             reason,
		ObservedGeneration: generation,
	}
}

// SameConditions reports whether two condition sets carry the same information,
// so an unchanged policy is not written back on every resync. Status writes are
// not free: a resync storm across every policy is a self-inflicted load spike
// on the API server.
func SameConditions(a, b []*metav1alpha1.DubboCondition) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].GetType() != b[i].GetType() ||
			a[i].GetStatus() != b[i].GetStatus() ||
			a[i].GetReason() != b[i].GetReason() ||
			a[i].GetObservedGeneration() != b[i].GetObservedGeneration() {
			return false
		}
	}
	return true
}

// Summary renders the conditions for logs and dubboctl output.
func Summary(conditions []*metav1alpha1.DubboCondition) string {
	parts := make([]string, 0, len(conditions))
	for _, item := range conditions {
		parts = append(parts, fmt.Sprintf("%s=%s(%s)", item.GetType(), item.GetStatus(), item.GetReason()))
	}
	return strings.Join(parts, " ")
}
