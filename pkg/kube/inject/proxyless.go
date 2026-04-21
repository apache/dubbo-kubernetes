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

	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/kdubbo/api/annotation"
	meshv1alpha1 "github.com/kdubbo/api/mesh/v1alpha1"
)

const (
	ProxylessGRPCTemplateName      = "grpc-engine"
	ProxylessXDSVolumeName         = "dubbo-xds"
	ProxylessXDSMountPath          = "/etc/dubbo/proxy"
	ProxylessGRPCBootstrapFileName = "grpc-bootstrap.json"
	ProxylessGRPCBootstrapPath     = ProxylessXDSMountPath + "/" + ProxylessGRPCBootstrapFileName
)

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

func ProxylessGRPCDiscoveryAddress(meshGlobalConfig *meshv1alpha1.MeshGlobalConfig, proxyConfig *meshv1alpha1.ProxyConfig) string {
	if meshGlobalConfig != nil {
		if cfg := meshGlobalConfig.GetDefaultConfig(); cfg != nil && cfg.GetDiscoveryAddress() != "" {
			return cfg.GetDiscoveryAddress()
		}
	}
	if proxyConfig != nil {
		return proxyConfig.GetDiscoveryAddress()
	}
	return ""
}

func ProxylessGRPCTrustDomain(meshGlobalConfig *meshv1alpha1.MeshGlobalConfig) string {
	if meshGlobalConfig != nil && meshGlobalConfig.GetTrustDomain() != "" {
		return meshGlobalConfig.GetTrustDomain()
	}
	return constants.DefaultClusterLocalDomain
}
