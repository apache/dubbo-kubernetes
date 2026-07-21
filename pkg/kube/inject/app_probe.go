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
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func FindProxy(pod *corev1.Pod) *corev1.Container {
	return FindContainerFromPod(ProxyContainerName, pod)
}

func FindContainerFromPod(name string, pod *corev1.Pod) *corev1.Container {
	if c := FindContainer(name, pod.Spec.Containers); c != nil {
		return c
	}
	return FindContainer(name, pod.Spec.Containers)
}

// RewriteAppProbes moves HTTP probes aimed at the application port onto the
// inbound sidecar port.
//
// The CNI rules REJECT every inbound TCP port of a managed Pod except the
// sidecar's, so the kubelet can never reach a probe pointed at the application
// port. The sidecar forwards its listen port to 127.0.0.1:<application port>,
// which means the rewritten probe still checks the same endpoint on the same
// container. Without this, application manifests have to hardcode the sidecar
// port and stop working outside the mesh.
//
// Only httpGet probes whose port resolves to the port the sidecar actually
// forwards to are touched; nothing forwards the other ports, so rewriting them
// would point the kubelet at an endpoint that does not exist.
func RewriteAppProbes(pod *corev1.Pod) {
	inbound := FindContainerFromPod(ProxylessGRPCInboundContainerName, pod)
	if inbound == nil {
		return
	}
	upstreamPort, found := inboundUpstreamPort(inbound)
	if !found {
		return
	}
	listenPort := inboundListenPort(inbound)
	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		if container.Name == ProxylessGRPCInboundContainerName {
			continue
		}
		for _, probe := range []*corev1.Probe{container.ReadinessProbe, container.LivenessProbe, container.StartupProbe} {
			rewriteProbePort(probe, container, upstreamPort, listenPort)
		}
	}
}

func rewriteProbePort(probe *corev1.Probe, container *corev1.Container, from, to int32) {
	if probe == nil || probe.HTTPGet == nil {
		return
	}
	port, resolved := resolveProbePort(probe.HTTPGet.Port, container)
	if !resolved || port != from {
		return
	}
	probe.HTTPGet.Port = intstr.FromInt32(to)
}

// resolveProbePort turns a probe port into a number, following the container's
// named ports when the probe refers to one.
func resolveProbePort(port intstr.IntOrString, container *corev1.Container) (int32, bool) {
	if port.Type == intstr.Int {
		return port.IntVal, true
	}
	for _, containerPort := range container.Ports {
		if containerPort.Name == port.StrVal {
			return containerPort.ContainerPort, true
		}
	}
	return 0, false
}

// inboundUpstreamPort reads the port the sidecar forwards to from its
// `--upstream <host>:<port>` argument. Reading it back from the rendered
// container keeps this in step with the template that produced it.
func inboundUpstreamPort(container *corev1.Container) (int32, bool) {
	if value, found := argValue(container.Args, "--upstream"); found {
		if _, portText, ok := strings.Cut(value, ":"); ok {
			if port, err := strconv.ParseInt(portText, 10, 32); err == nil {
				return int32(port), true
			}
		}
	}
	return 0, false
}

func inboundListenPort(container *corev1.Container) int32 {
	if value, found := argValue(container.Args, "--listen"); found {
		if _, portText, ok := strings.Cut(value, ":"); ok {
			if port, err := strconv.ParseInt(portText, 10, 32); err == nil {
				return int32(port)
			}
		}
	}
	return ProxylessGRPCInboundPort
}

// argValue supports both `--flag value` and `--flag=value`.
func argValue(args []string, name string) (string, bool) {
	for i, arg := range args {
		if arg == name && i+1 < len(args) {
			return args[i+1], true
		}
		if value, found := strings.CutPrefix(arg, name+"="); found {
			return value, true
		}
	}
	return "", false
}
