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
	"strings"

	"github.com/kdubbo/api/annotation"
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
	// InherentGatewayInboundPort is the managed dxgate listener. Application
	// workloads keep their own declared ports in proxyless mode.
	InherentGatewayInboundPort = 15080
)

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
