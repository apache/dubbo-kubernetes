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

package inject

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func httpProbe(port intstr.IntOrString) *corev1.Probe {
	return &corev1.Probe{ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{
		Path: "/healthz",
		Port: port,
	}}}
}

func inboundContainer(listen, upstream string) corev1.Container {
	return corev1.Container{
		Name:  ProxylessGRPCInboundContainerName,
		Image: "kdubbo/dubbod:latest",
		Args:  []string{"grpc-inbound", "--listen", listen, "--upstream", upstream},
		Ports: []corev1.ContainerPort{{Name: "grpc-inbound", ContainerPort: ProxylessGRPCInboundPort}},
	}
}

func appContainer(probes ...*corev1.Probe) corev1.Container {
	c := corev1.Container{
		Name:  "app",
		Image: "example/app:latest",
		Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: 9080}},
	}
	if len(probes) > 0 {
		c.ReadinessProbe = probes[0]
	}
	if len(probes) > 1 {
		c.LivenessProbe = probes[1]
	}
	if len(probes) > 2 {
		c.StartupProbe = probes[2]
	}
	return c
}

func podWith(containers ...corev1.Container) *corev1.Pod {
	return &corev1.Pod{Spec: corev1.PodSpec{Containers: containers}}
}

// The point of the rewrite: a manifest that names its own port keeps working once
// the CNI rules make that port unreachable from the kubelet.
func TestRewriteAppProbesMovesApplicationPortToSidecar(t *testing.T) {
	pod := podWith(
		appContainer(httpProbe(intstr.FromInt32(9080)), httpProbe(intstr.FromInt32(9080)), httpProbe(intstr.FromInt32(9080))),
		inboundContainer(":15080", "127.0.0.1:9080"),
	)
	RewriteAppProbes(pod)
	app := pod.Spec.Containers[0]
	for name, probe := range map[string]*corev1.Probe{
		"readiness": app.ReadinessProbe,
		"liveness":  app.LivenessProbe,
		"startup":   app.StartupProbe,
	} {
		if got := probe.HTTPGet.Port.IntValue(); got != ProxylessGRPCInboundPort {
			t.Fatalf("%s probe port = %d, want %d", name, got, ProxylessGRPCInboundPort)
		}
		if probe.HTTPGet.Path != "/healthz" {
			t.Fatalf("%s probe path changed to %q", name, probe.HTTPGet.Path)
		}
	}
}

func TestRewriteAppProbesResolvesNamedPort(t *testing.T) {
	pod := podWith(
		appContainer(httpProbe(intstr.FromString("http"))),
		inboundContainer(":15080", "127.0.0.1:9080"),
	)
	RewriteAppProbes(pod)
	if got := pod.Spec.Containers[0].ReadinessProbe.HTTPGet.Port.IntValue(); got != ProxylessGRPCInboundPort {
		t.Fatalf("port = %d, want %d", got, ProxylessGRPCInboundPort)
	}
}

// Nothing forwards other ports, so pointing the kubelet at the sidecar for them
// would break a probe that the mesh happens to leave working.
func TestRewriteAppProbesLeavesOtherPortsAlone(t *testing.T) {
	pod := podWith(
		appContainer(httpProbe(intstr.FromInt32(8081))),
		inboundContainer(":15080", "127.0.0.1:9080"),
	)
	RewriteAppProbes(pod)
	if got := pod.Spec.Containers[0].ReadinessProbe.HTTPGet.Port.IntValue(); got != 8081 {
		t.Fatalf("port = %d, want 8081 unchanged", got)
	}
}

func TestRewriteAppProbesIgnoresNonHTTPProbes(t *testing.T) {
	app := appContainer()
	app.ReadinessProbe = &corev1.Probe{ProbeHandler: corev1.ProbeHandler{
		TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt32(9080)},
	}}
	app.LivenessProbe = &corev1.Probe{ProbeHandler: corev1.ProbeHandler{
		Exec: &corev1.ExecAction{Command: []string{"true"}},
	}}
	pod := podWith(app, inboundContainer(":15080", "127.0.0.1:9080"))
	RewriteAppProbes(pod)
	if got := pod.Spec.Containers[0].ReadinessProbe.TCPSocket.Port.IntValue(); got != 9080 {
		t.Fatalf("tcpSocket port = %d, want 9080 unchanged", got)
	}
	if pod.Spec.Containers[0].LivenessProbe.Exec == nil {
		t.Fatal("exec probe was modified")
	}
}

// An uninjected Pod must come out untouched, otherwise the probe would point at a
// port no container listens on.
func TestRewriteAppProbesNoopWithoutSidecar(t *testing.T) {
	pod := podWith(appContainer(httpProbe(intstr.FromInt32(9080))))
	RewriteAppProbes(pod)
	if got := pod.Spec.Containers[0].ReadinessProbe.HTTPGet.Port.IntValue(); got != 9080 {
		t.Fatalf("port = %d, want 9080 unchanged", got)
	}
}

func TestRewriteAppProbesNoopWhenUpstreamUnknown(t *testing.T) {
	inbound := inboundContainer(":15080", "127.0.0.1:9080")
	inbound.Args = []string{"grpc-inbound", "--listen", ":15080"}
	pod := podWith(appContainer(httpProbe(intstr.FromInt32(9080))), inbound)
	RewriteAppProbes(pod)
	if got := pod.Spec.Containers[0].ReadinessProbe.HTTPGet.Port.IntValue(); got != 9080 {
		t.Fatalf("port = %d, want 9080 unchanged", got)
	}
}

func TestRewriteAppProbesFollowsSidecarListenPort(t *testing.T) {
	pod := podWith(
		appContainer(httpProbe(intstr.FromInt32(9080))),
		inboundContainer(":15081", "127.0.0.1:9080"),
	)
	RewriteAppProbes(pod)
	if got := pod.Spec.Containers[0].ReadinessProbe.HTTPGet.Port.IntValue(); got != 15081 {
		t.Fatalf("port = %d, want 15081", got)
	}
}

func TestRewriteAppProbesAcceptsEqualsSyntax(t *testing.T) {
	inbound := inboundContainer(":15080", "127.0.0.1:9080")
	inbound.Args = []string{"grpc-inbound", "--listen=:15080", "--upstream=127.0.0.1:9080"}
	pod := podWith(appContainer(httpProbe(intstr.FromInt32(9080))), inbound)
	RewriteAppProbes(pod)
	if got := pod.Spec.Containers[0].ReadinessProbe.HTTPGet.Port.IntValue(); got != ProxylessGRPCInboundPort {
		t.Fatalf("port = %d, want %d", got, ProxylessGRPCInboundPort)
	}
}

// Running twice must not move an already-rewritten probe again.
func TestRewriteAppProbesIsIdempotent(t *testing.T) {
	pod := podWith(
		appContainer(httpProbe(intstr.FromInt32(9080))),
		inboundContainer(":15080", "127.0.0.1:9080"),
	)
	RewriteAppProbes(pod)
	RewriteAppProbes(pod)
	if got := pod.Spec.Containers[0].ReadinessProbe.HTTPGet.Port.IntValue(); got != ProxylessGRPCInboundPort {
		t.Fatalf("port = %d, want %d", got, ProxylessGRPCInboundPort)
	}
}

func TestRewriteAppProbesLeavesSidecarProbeAlone(t *testing.T) {
	inbound := inboundContainer(":15080", "127.0.0.1:9080")
	inbound.ReadinessProbe = httpProbe(intstr.FromInt32(9080))
	pod := podWith(appContainer(), inbound)
	RewriteAppProbes(pod)
	if got := pod.Spec.Containers[1].ReadinessProbe.HTTPGet.Port.IntValue(); got != 9080 {
		t.Fatalf("sidecar probe port = %d, want 9080 unchanged", got)
	}
}
