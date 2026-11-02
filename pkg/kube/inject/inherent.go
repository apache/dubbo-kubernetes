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

	"github.com/dubml/api/annotation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	InherentTripleTemplateName         = "triple-engine"
	InherentTripleBootstrapEnvName     = "TRIPLE_XDS_BOOTSTRAP"
	InherentTripleBootstrapFileName    = "triple-bootstrap.json"
	InherentTripleBootstrapPath        = InherentXDSMountPath + "/" + InherentTripleBootstrapFileName
	InherentTripleStatusAddressEnvName = "DUBBO_TRIPLE_STATUS_ADDRESS"
	InherentTripleStatusAddress        = "0.0.0.0:9090"
	InherentXDSVolumeName              = "dubbo-xds"
	InherentXDSMountPath               = "/etc/dubbo/proxy"
	InherentMetricsPortName            = "metrics"
	InherentMetricsPort                = 9090
	// InherentGatewayInboundPort is the managed gateway listener. Application
	// workloads keep their own declared ports in proxyless mode.
	InherentGatewayInboundPort = 15080
)

var InherentInjectTemplatesAnnoName = annotation.OrgApacheDubboInjectTemplates.Name

func InherentXDSSecretName(podName string) string {
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

func InherentXDSSecretNameForMeta(meta metav1.ObjectMeta) string {
	return InherentXDSSecretName(meta.Name)
}

func inherentXDSPodName(prefix string, requestUID types.UID) string {
	const suffixLength = 16
	if len(prefix) > 63-suffixLength {
		prefix = prefix[:63-suffixLength]
	}
	sum := sha256.Sum256([]byte(requestUID))
	return prefix + hex.EncodeToString(sum[:suffixLength/2])
}
