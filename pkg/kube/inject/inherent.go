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
	InherentGRPCTemplateName                    = "grpc-engine"
	InherentXDSVolumeName                       = "dubbo-xds"
	InherentXDSMountPath                        = "/etc/dubbo/proxy"
	InherentGRPCBootstrapFileName               = "grpc-bootstrap.json"
	InherentGRPCBootstrapPath                   = InherentXDSMountPath + "/" + InherentGRPCBootstrapFileName
	InherentXDSAddressEnvName                   = "XDS_ADDRESS"
	InherentGRPCConfigEnvName                   = "DUBBO_GRPC_XDS_CONFIG"
	InherentGRPCMetricsAddressEnvName           = "DUBBO_GRPC_METRICS_ADDRESS"
	InherentGRPCMetricsAddress                  = ":9090"
	InherentGRPCMetricsPortName                 = "metrics"
	InherentGRPCMetricsPort                     = 9090
	InherentGRPCKeepaliveEnvName                = "DUBBO_GRPC_KEEPALIVE"
	InherentGRPCKeepaliveTimeEnv                = "GRPC_KEEPALIVE_INTERVAL"
	InherentGRPCKeepaliveTimeoutEnv             = "GRPC_KEEPALIVE_TIMEOUT"
	InherentGRPCKeepalivePermitWithoutStreamEnv = "GRPC_KEEPALIVE_PERMIT_WITHOUT_STREAM"
	InherentGRPCKeepaliveValue                  = "true"
	InherentGRPCKeepaliveTime                   = "30s"
	InherentGRPCKeepaliveTimeout                = "10s"
	InherentGRPCConfigFileName                  = "dubbo-grpc-xds.json"
	InherentGRPCConfigPath                      = InherentXDSMountPath + "/" + InherentGRPCConfigFileName
	InherentGRPCInboundContainerName            = "dubbo-grpc-inbound"
	InherentGRPCInboundPort                     = 15080
	// InherentGRPCInboundAdminPort serves the inbound sidecar's health,
	// readiness and metrics endpoints. It carries no mesh traffic, so the node
	// fence exempts it and kubelet can probe it directly.
	InherentGRPCInboundAdminPort = 15020
	InherentManagedLabel         = "inherent.dubbo.apache.org/managed"
	InherentManagedLabelValue    = "true"
)

// InherentExcludeInboundPortsAnnotation lists inbound ports that stay
// reachable without passing through the mTLS listener, as a comma-separated
// list of port numbers. Setting it replaces the default, which is every
// declared container port that the inbound listener does not forward to.
const InherentExcludeInboundPortsAnnotation = "inherent.dubbo.apache.org/excludeInboundPorts"

// InherentExcludedInboundPorts reports the ports that must be exempted from
// the inbound fence for a pod.
//
// The inbound listener forwards a single upstream port, but a workload
// commonly declares more: metrics, admin, or debug endpoints. Those are not
// part of the mesh, and without an exemption the node fence rejects them,
// which strands them with no diagnosable symptom. Excluded ports carry plain
// traffic and are not covered by mTLS.
func InherentExcludedInboundPorts(pod *corev1.Pod) ([]int, error) {
	if pod == nil {
		return nil, nil
	}
	if raw, ok := pod.Annotations[InherentExcludeInboundPortsAnnotation]; ok {
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
			return nil, fmt.Errorf("annotation %s: %q is not a port number", InherentExcludeInboundPortsAnnotation, field)
		}
		if port < 1 || port > 65535 {
			return nil, fmt.Errorf("annotation %s: port %d is out of range", InherentExcludeInboundPortsAnnotation, port)
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
	if proxy := FindContainerFromPod(InherentGRPCInboundContainerName, pod); proxy != nil {
		meshed[inboundListenPort(proxy)] = struct{}{}
		if upstream, ok := inboundUpstreamPort(proxy); ok {
			meshed[upstream] = struct{}{}
		}
	} else {
		meshed[InherentGRPCInboundPort] = struct{}{}
	}

	seen := map[int]struct{}{}
	ports := []int{}
	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		if container.Name == InherentGRPCInboundContainerName {
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

var InherentInjectTemplatesAnnoName = annotation.OrgApacheDubboInjectTemplates.Name

func InherentGRPCSecretName(podName string) string {
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

func InherentGRPCSecretNameForMeta(meta metav1.ObjectMeta) string {
	name := meta.Name
	if meta.GenerateName != "" {
		name = meta.GenerateName
	}
	return InherentGRPCSecretName(name)
}
