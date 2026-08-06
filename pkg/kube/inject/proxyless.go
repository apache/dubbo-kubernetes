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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/kdubbo/api/annotation"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ProxylessGRPCTemplateName                    = "grpc-engine"
	ProxylessXDSVolumeName                       = "dubbo-xds"
	ProxylessXDSMountPath                        = "/etc/dubbo/proxy"
	ProxylessGRPCBootstrapFileName               = "grpc-bootstrap.json"
	ProxylessGRPCBootstrapPath                   = ProxylessXDSMountPath + "/" + ProxylessGRPCBootstrapFileName
	ProxylessXDSAddressEnvName                   = "XDS_ADDRESS"
	ProxylessGRPCConfigEnvName                   = "DUBBO_GRPC_XDS_CONFIG"
	ProxylessGRPCKeepaliveEnvName                = "DUBBO_GRPC_KEEPALIVE"
	ProxylessGRPCKeepaliveTimeEnv                = "GRPC_KEEPALIVE_INTERVAL"
	ProxylessGRPCKeepaliveTimeoutEnv             = "GRPC_KEEPALIVE_TIMEOUT"
	ProxylessGRPCKeepalivePermitWithoutStreamEnv = "GRPC_KEEPALIVE_PERMIT_WITHOUT_STREAM"
	ProxylessGRPCKeepaliveValue                  = "true"
	ProxylessGRPCKeepaliveTime                   = "30s"
	ProxylessGRPCKeepaliveTimeout                = "10s"
	ProxylessGRPCConfigFileName                  = "dubbo-grpc-xds.json"
	ProxylessGRPCConfigPath                      = ProxylessXDSMountPath + "/" + ProxylessGRPCConfigFileName
	ProxylessGRPCInboundContainerName            = "dubbo-grpc-inbound"
	ProxylessGRPCInboundPort                     = 15080
	// ProxylessGRPCInboundAdminPort serves the inbound sidecar's health,
	// readiness and metrics endpoints. It carries no mesh traffic, so the node
	// fence exempts it and kubelet can probe it directly.
	ProxylessGRPCInboundAdminPort = 15020
	ProxylessManagedLabel         = "proxyless.dubbo.apache.org/managed"
	ProxylessManagedLabelValue    = "true"
)

// ProxylessExcludeInboundPortsAnnotation lists inbound ports that stay
// reachable without passing through the mTLS listener, as a comma-separated
// list of port numbers. Setting it replaces the default, which is every
// declared container port that the inbound listener does not forward to.
const ProxylessExcludeInboundPortsAnnotation = "proxyless.dubbo.apache.org/excludeInboundPorts"

// ProxylessExcludedInboundPorts reports the ports that must be exempted from
// the inbound fence for a pod.
//
// The inbound listener forwards a single upstream port, but a workload
// commonly declares more: metrics, admin, or debug endpoints. Those are not
// part of the mesh, and without an exemption the node fence rejects them,
// which strands them with no diagnosable symptom. Excluded ports carry plain
// traffic and are not covered by mTLS.
func ProxylessExcludedInboundPorts(pod *corev1.Pod) ([]int, error) {
	if pod == nil {
		return nil, nil
	}
	if raw, ok := pod.Annotations[ProxylessExcludeInboundPortsAnnotation]; ok {
		return parseExcludedInboundPorts(raw)
	}
	return defaultExcludedInboundPorts(pod), nil
}

func parseExcludedInboundPorts(raw string) ([]int, error) {
	seen := map[int]struct{}{}
	ports := []int{}
	for _, field := range strings.Split(raw, ",") {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		port, err := strconv.Atoi(field)
		if err != nil {
			return nil, fmt.Errorf("annotation %s: %q is not a port number", ProxylessExcludeInboundPortsAnnotation, field)
		}
		if port < 1 || port > 65535 {
			return nil, fmt.Errorf("annotation %s: port %d is out of range", ProxylessExcludeInboundPortsAnnotation, port)
		}
		if _, ok := seen[port]; ok {
			continue
		}
		seen[port] = struct{}{}
		ports = append(ports, port)
	}
	sort.Ints(ports)
	return ports, nil
}

// defaultExcludedInboundPorts exempts every declared container port that the
// inbound listener neither listens on nor forwards to.
func defaultExcludedInboundPorts(pod *corev1.Pod) []int {
	meshed := map[int32]struct{}{}
	if proxy := FindContainerFromPod(ProxylessGRPCInboundContainerName, pod); proxy != nil {
		meshed[inboundListenPort(proxy)] = struct{}{}
		if upstream, ok := inboundUpstreamPort(proxy); ok {
			meshed[upstream] = struct{}{}
		}
	} else {
		meshed[ProxylessGRPCInboundPort] = struct{}{}
	}

	seen := map[int]struct{}{}
	ports := []int{}
	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		if container.Name == ProxylessGRPCInboundContainerName {
			continue
		}
		for _, port := range container.Ports {
			if port.ContainerPort == 0 {
				continue
			}
			if _, ok := meshed[port.ContainerPort]; ok {
				continue
			}
			value := int(port.ContainerPort)
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			ports = append(ports, value)
		}
	}
	sort.Ints(ports)
	return ports
}

var ProxylessInjectTemplatesAnnoName = annotation.OrgApacheDubboInjectTemplates.Name

func ProxylessGRPCSecretName(podName string) string {
	const (
		prefix      = "dubbo-xds-"
		maxNameLen  = 63
		hashHexSize = 8
	)

	sum := sha256.Sum256([]byte(podName))
	suffix := hex.EncodeToString(sum[:hashHexSize/2])
	baseMaxLen := maxNameLen - len(prefix) - 1 - len(suffix)
	base := podName
	if len(base) > baseMaxLen {
		base = base[:baseMaxLen]
	}
	base = strings.Trim(base, "-")
	if base == "" {
		base = "pod"
	}

	return fmt.Sprintf("%s%s-%s", prefix, base, suffix)
}

func ProxylessGRPCSecretNameForMeta(meta metav1.ObjectMeta) string {
	name := meta.Name
	if meta.GenerateName != "" {
		name = meta.GenerateName
	}
	return ProxylessGRPCSecretName(name)
}
