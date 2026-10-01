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

package bootstrap

import (
	"fmt"
	"strings"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/server/ca/authenticate"
	"github.com/apache/dubbo-kubernetes/pkg/security"
	"github.com/apache/dubbo-kubernetes/pkg/spiffe"
	corev1 "k8s.io/api/core/v1"
)

func (s *Server) initXDSAuthentication() {
	s.XDSServer.Authenticators = []security.Authenticator{&authenticate.ClientCertAuthenticator{}}
	s.XDSServer.Authorize = s.authorizeXDSWorkload
}

func (s *Server) authorizeXDSWorkload(proxy *model.Proxy, identities []string) error {
	if proxy == nil || proxy.Metadata == nil || proxy.Metadata.Namespace == "" || proxy.Metadata.ClusterID == "" {
		return fmt.Errorf("xDS requires a workload namespace and cluster")
	}
	controller := s.xdsWorkloadController(proxy)
	if controller == nil {
		return fmt.Errorf("xDS workload cluster is not managed")
	}
	suffix := "." + proxy.Metadata.Namespace
	if !strings.HasSuffix(proxy.ID, suffix) {
		return fmt.Errorf("xDS node namespace does not match metadata")
	}
	pod := controller.pods.Get(strings.TrimSuffix(proxy.ID, suffix), proxy.Metadata.Namespace)
	mesh := s.environment.Mesh()
	trustDomains := append([]string{mesh.GetTrustDomain()}, mesh.GetTrustDomainAliases()...)
	return authorizeXDSPod(proxy, identities, pod, trustDomains)
}

func (s *Server) xdsWorkloadController(proxy *model.Proxy) *inherentGRPCWorkloadController {
	if proxy.Metadata.ClusterID != s.clusterID {
		return nil
	}
	return s.inherentGRPCWorkloadController
}

func authorizeXDSPod(proxy *model.Proxy, identities []string, pod *corev1.Pod, trustDomains []string) error {
	if pod == nil || pod.DeletionTimestamp != nil || !shouldManageInherentGRPCPod(pod) {
		return fmt.Errorf("xDS node is not an active managed workload")
	}
	if proxy.Metadata.Namespace != pod.Namespace || proxy.ID != pod.Name+"."+pod.Namespace {
		return fmt.Errorf("xDS node does not match workload")
	}
	if len(proxy.IPAddresses) != 1 || !podHasIP(pod, proxy.IPAddresses[0]) {
		return fmt.Errorf("xDS node IP does not match workload")
	}
	if proxy.IsRouter() && pod.Labels["gateway.networking.k8s.io/gateway-name"] == "" {
		return fmt.Errorf("xDS router is not a managed gateway")
	}
	serviceAccount := pod.Spec.ServiceAccountName
	if serviceAccount == "" {
		serviceAccount = "default"
	}
	for _, raw := range identities {
		identity, err := spiffe.ParseIdentity(raw)
		if err != nil || identity.Namespace != pod.Namespace || identity.ServiceAccount != serviceAccount {
			continue
		}
		for _, trustDomain := range trustDomains {
			if identity.TrustDomain == trustDomain {
				return nil
			}
		}
	}
	return fmt.Errorf("xDS identity does not match workload service account and trust domain")
}

func podHasIP(pod *corev1.Pod, ip string) bool {
	if ip != "" && ip == pod.Status.PodIP {
		return true
	}
	for _, address := range pod.Status.PodIPs {
		if ip != "" && address.IP == ip {
			return true
		}
	}
	return false
}
