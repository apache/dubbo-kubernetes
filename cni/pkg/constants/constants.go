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

package constants

const (
	// PluginName is the CNI plugin name registered in chained CNI conflist files.
	PluginName = "dubbo-cni"

	// DefaultCNIBinDir is the host CNI plugin binary directory.
	DefaultCNIBinDir = "/opt/cni/bin"
	// DefaultCNINetDir is the host CNI network configuration directory.
	DefaultCNINetDir = "/etc/cni/net.d"
	// DefaultPluginBinarySource is the container path to the bundled dubbo-cni binary.
	DefaultPluginBinarySource = "/usr/local/bin/dubbo-cni"
	// DefaultSocketPath is the Unix domain socket shared between dubbo-cni and dubbo-cni-node.
	DefaultSocketPath = "/var/run/dubbo-cni/cni.sock"

	// OutboundCapturePort is the node-local transparent outbound proxy listener port.
	OutboundCapturePort = 15001
	// InboundTunnelPort is the node-level mTLS HTTP/2 CONNECT tunnel listener port.
	InboundTunnelPort = 15008
	// InboundPlaintextPort is the node-local inbound transparent capture port.
	InboundPlaintextPort = 15006
	// DefaultStatusPort is the HTTP port for /healthz, /readyz, /debug/pods, /debug/syncz, /metrics.
	DefaultStatusPort = 15021
	// DefaultProxyUID is the UID excluded from iptables redirection to prevent loops.
	DefaultProxyUID = 2509
	// DefaultDubbodXDSPort is dubbod's secure xDS port excluded from outbound redirection.
	DefaultDubbodXDSPort = 26012

	// RedirectionAnnotation marks pods enrolled in CNI node-level redirection.
	RedirectionAnnotation = "cni.dubbo.apache.org/redirection"
	// RedirectionEnabledValue is the enabled value for RedirectionAnnotation.
	RedirectionEnabledValue = "enabled"
	// RedirectionDisabledValue explicitly opts a pod out of CNI redirection.
	RedirectionDisabledValue = "disabled"

	// TunnelALPN is the ALPN protocol negotiated on node-to-node Triple/HBONE mTLS tunnels.
	TunnelALPN = "h2"

	// EnvNodeName supplies the Kubernetes Node name to dubbo-cni-node.
	EnvNodeName = "NODE_NAME"
	// EnvSystemNamespace supplies the Dubbo control-plane namespace.
	EnvSystemNamespace = "DUBBO_SYSTEM_NAMESPACE"
	// EnvDiscoveryAddress overrides the dubbod xDS discovery address.
	EnvDiscoveryAddress = "XDS_ADDRESS"
)
