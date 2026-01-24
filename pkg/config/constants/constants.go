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

package constants

const (
	UnspecifiedIP = "0.0.0.0"

	DubboSystemNamespace = "dubbo-system"
	KubeSystemNamespace  = "kube-system"

	DefaultClusterLocalDomain = "cluster.local"
	DefaultClusterName        = "Kubernetes"

	CertChainFilename = "cert-chain.pem"

	CertProviderDubbod                 = "dubbod"
	CertProviderKubernetes             = "kubernetes"
	CertProviderKubernetesSignerPrefix = "k8s.io/"
	CertProviderNone                   = "none"
	CertProviderCustom                 = "custom"

	CACertNamespaceConfigMapDataName = "root-cert.pem"
	CACRLNamespaceConfigMapDataName  = "ca-crl.pem"

	PodInfoAnnotationsPath = "./etc/dubbo/pod/annotations"

	StatPrefixDelimiter = ";"

	DubboWellKnownDNSCertPath   = "./var/run/secrets/dubbod/tls/"
	DubboWellKnownDNSCaCertPath = "./var/run/secrets/dubbod/ca/"

	ConfigPathDir                      = "./etc/dubbo/proxy"
	KeyFilename                        = "key.pem"
	DefaultDubboTLSCert                = DubboWellKnownDNSCertPath + "tls.crt"
	DefaultDubboTLSKey                 = DubboWellKnownDNSCertPath + "tls.key"
	DefaultDubboTLSCaCert              = DubboWellKnownDNSCaCertPath + "root-cert.pem"
	DefaultDubboTLSCaCertAlternatePath = DubboWellKnownDNSCertPath + "ca.crt"

	AlwaysReject = "internal.dubbo.apache.org/webhook-always-reject"

	ManagedGatewayControllerLabel = "dubbo.apache.org-gateway-controller"
)
