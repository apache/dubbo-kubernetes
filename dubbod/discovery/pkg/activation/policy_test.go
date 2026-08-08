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
	"context"
	"testing"
	"time"

	networking "github.com/kdubbo/api/networking/v1alpha3"
	clientnetworking "github.com/kdubbo/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type services map[string]bool

func (s services) HasService(namespace, name string) bool { return s[namespace+"/"+name] }

type scalerStatus bool

func (s scalerStatus) ScalerReady(*clientnetworking.ServiceActivationPolicy) bool { return bool(s) }

type activatorStatus bool

func (a activatorStatus) ActivatorReady(*clientnetworking.ServiceActivationPolicy) bool {
	return bool(a)
}

// policy builds the kube wrapper from its parts. The generated spec is a proto
// message with an embedded mutex, so it is assembled in place rather than
// copied in from a variable.
func policy(
	target *networking.PolicyTargetReference,
	autoscaler *networking.AutoscalerReference,
	protocols ...networking.ActivationProtocol,
) *clientnetworking.ServiceActivationPolicy {
	return &clientnetworking.ServiceActivationPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "orders", Namespace: "app", Generation: 7},
		Spec: networking.ServiceActivationPolicy{
			TargetRef:     target,
			AutoscalerRef: autoscaler,
			Protocols:     protocols,
		},
	}
}

func serviceTarget(name string) *networking.PolicyTargetReference {
	return &networking.PolicyTargetReference{Kind: "Service", Name: name}
}

func autoscaler(name string) *networking.AutoscalerReference {
	return &networking.AutoscalerReference{Name: name}
}

func validPolicy() *clientnetworking.ServiceActivationPolicy {
	return policy(serviceTarget("orders"), autoscaler("orders"))
}

func conditionsByType(t *testing.T, evaluator PolicyEvaluator, p *clientnetworking.ServiceActivationPolicy) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, item := range evaluator.Evaluate(p) {
		if item.GetObservedGeneration() != p.GetGeneration() {
			t.Fatalf("%s observedGeneration = %d, want %d",
				item.GetType(), item.GetObservedGeneration(), p.GetGeneration())
		}
		out[item.GetType()] = item.GetStatus() + "/" + item.GetReason()
	}
	return out
}

func TestEvaluateReportsEveryConditionTrueWhenThePathIsComplete(t *testing.T) {
	evaluator := PolicyEvaluator{
		Services:  services{"app/orders": true},
		Scaler:    scalerStatus(true),
		Activator: activatorStatus(true),
	}

	got := conditionsByType(t, evaluator, validPolicy())
	for _, conditionType := range []string{
		ConditionAccepted, ConditionEligible, ConditionScalerReady, ConditionActivatorReady,
	} {
		if status := got[conditionType]; status[:4] != "True" {
			t.Fatalf("%s = %q, want True", conditionType, status)
		}
	}
}

func TestEvaluateRejectsMalformedPolicies(t *testing.T) {
	evaluator := PolicyEvaluator{Services: services{"app/orders": true}}

	tests := []struct {
		name       string
		target     *networking.PolicyTargetReference
		autoscaler *networking.AutoscalerReference
		reason     string
	}{
		{
			name:       "no target",
			autoscaler: autoscaler("orders"),
			reason:     "TargetRefMissing",
		},
		{
			name:       "blank target name",
			target:     serviceTarget("  "),
			autoscaler: autoscaler("orders"),
			reason:     "TargetRefMissing",
		},
		{
			// Only Services have endpoints to wait on. Anything else leaves the
			// gateway holding requests for something that never becomes routable.
			name:       "non-Service target",
			target:     &networking.PolicyTargetReference{Kind: "Deployment", Name: "orders"},
			autoscaler: autoscaler("orders"),
			reason:     "TargetKindUnsupported",
		},
		{
			name:   "no autoscaler",
			target: serviceTarget("orders"),
			reason: "AutoscalerRefMissing",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := conditionsByType(t, evaluator, policy(test.target, test.autoscaler))
			if want := "False/" + test.reason; got[ConditionAccepted] != want {
				t.Fatalf("Accepted = %q, want %q", got[ConditionAccepted], want)
			}
			// The rest of the path does not exist yet; reporting on it would
			// invent answers about a target that was never resolved.
			for _, conditionType := range []string{
				ConditionEligible, ConditionScalerReady, ConditionActivatorReady,
			} {
				if want := "False/PolicyNotAccepted"; got[conditionType] != want {
					t.Fatalf("%s = %q, want %q", conditionType, got[conditionType], want)
				}
			}
		})
	}
}

func TestEvaluateReportsMissingTargetService(t *testing.T) {
	evaluator := PolicyEvaluator{Services: services{}}
	got := conditionsByType(t, evaluator, validPolicy())
	if want := "False/TargetServiceNotFound"; got[ConditionAccepted] != want {
		t.Fatalf("Accepted = %q, want %q", got[ConditionAccepted], want)
	}
}

// A stream cannot be replayed once the backend is up, so a policy naming a
// streaming protocol must not look ready.
func TestEvaluateRejectsProtocolsThatCannotBeHeld(t *testing.T) {
	evaluator := PolicyEvaluator{
		Services:  services{"app/orders": true},
		Scaler:    scalerStatus(true),
		Activator: activatorStatus(true),
	}

	unknown := policy(serviceTarget("orders"), autoscaler("orders"), networking.ActivationProtocol(99))

	got := conditionsByType(t, evaluator, unknown)
	if got[ConditionAccepted][:4] != "True" {
		t.Fatalf("Accepted = %q, want True", got[ConditionAccepted])
	}
	if want := "False/ProtocolNotActivatable"; got[ConditionEligible] != want {
		t.Fatalf("Eligible = %q, want %q", got[ConditionEligible], want)
	}
}

func TestEvaluateAcceptsEveryActivatableProtocol(t *testing.T) {
	evaluator := PolicyEvaluator{Services: services{"app/orders": true}}
	all := policy(serviceTarget("orders"), autoscaler("orders"),
		networking.ActivationProtocol_HTTP,
		networking.ActivationProtocol_GRPC_UNARY,
		networking.ActivationProtocol_TRIPLE_UNARY,
	)

	got := conditionsByType(t, evaluator, all)
	if want := "True/Eligible"; got[ConditionEligible] != want {
		t.Fatalf("Eligible = %q, want %q", got[ConditionEligible], want)
	}
}

// A ScaledObject that never points its trigger here looks identical to a
// working one until the first request is dropped, so it has to be reported.
func TestEvaluateReportsScalerAndActivatorSeparately(t *testing.T) {
	evaluator := PolicyEvaluator{
		Services:  services{"app/orders": true},
		Scaler:    scalerStatus(false),
		Activator: activatorStatus(true),
	}
	got := conditionsByType(t, evaluator, validPolicy())
	if want := "False/ScaledObjectNotReady"; got[ConditionScalerReady] != want {
		t.Fatalf("ScalerReady = %q, want %q", got[ConditionScalerReady], want)
	}
	if want := "True/GatewayProgrammed"; got[ConditionActivatorReady] != want {
		t.Fatalf("ActivatorReady = %q, want %q", got[ConditionActivatorReady], want)
	}

	evaluator = PolicyEvaluator{
		Services:  services{"app/orders": true},
		Scaler:    scalerStatus(true),
		Activator: activatorStatus(false),
	}
	got = conditionsByType(t, evaluator, validPolicy())
	if want := "True/ScaledObjectReady"; got[ConditionScalerReady] != want {
		t.Fatalf("ScalerReady = %q, want %q", got[ConditionScalerReady], want)
	}
	if want := "False/NoProgrammedGateway"; got[ConditionActivatorReady] != want {
		t.Fatalf("ActivatorReady = %q, want %q", got[ConditionActivatorReady], want)
	}
}

// Evaluate feeds a change detector, so an unchanged policy must produce an
// identical result; otherwise every resync writes status back to the API server.
func TestEvaluateIsStableAcrossCalls(t *testing.T) {
	evaluator := PolicyEvaluator{
		Services:  services{"app/orders": true},
		Scaler:    scalerStatus(true),
		Activator: activatorStatus(true),
	}
	target := validPolicy()

	if !SameConditions(evaluator.Evaluate(target), evaluator.Evaluate(target)) {
		t.Fatal("Evaluate() produced different conditions for an unchanged policy")
	}

	// A real change must still be detected.
	changed := PolicyEvaluator{
		Services:  services{"app/orders": true},
		Scaler:    scalerStatus(false),
		Activator: activatorStatus(true),
	}
	if SameConditions(evaluator.Evaluate(target), changed.Evaluate(target)) {
		t.Fatal("SameConditions() reported no change after the scaler unsubscribed")
	}
}

// ScalerReady is derived from live KEDA streams, so the tracking has to follow
// the stream's lifetime exactly: a policy must stop looking ready the moment
// KEDA stops listening.
func TestScalerTracksSubscriptionForTheController(t *testing.T) {
	scaler := NewScaler(NewRegistry())
	if scaler.Subscribed(orders) {
		t.Fatal("Subscribed() = true before any stream opened")
	}

	ctx, cancel := context.WithCancel(context.Background())
	stream := newFakeStream(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = scaler.StreamIsActive(serviceRef(), stream)
	}()

	// The first send happens after the subscription is registered, so receiving
	// it means tracking is in place.
	stream.next(t)
	if !scaler.Subscribed(orders) {
		t.Fatal("Subscribed() = false while a stream is open")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("StreamIsActive did not return after the context was canceled")
	}
	if scaler.Subscribed(orders) {
		t.Fatal("Subscribed() = true after the stream closed")
	}
}
