/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package constants

const (
	UnspecifiedIP = "0.0.0.0"

	DubboSystemNamespace               = "dubbo-system"
	DefaultClusterLocalDomain          = "cluster.local"
	DefaultClusterName                 = "Kubernetes"
	ServiceClusterName                 = "dubbo-proxy"
	ConfigPathDir                      = "./etc/dubbo/proxy"
	BinaryPathFilename                 = "/usr/local/bin/sail-agent"
	KeyFilename                        = "key.pem"
	CertChainFilename                  = "cert-chain.pem"
	CertProviderDubbod                 = "dubbod"
	CertProviderKubernetes             = "kubernetes"
	CertProviderKubernetesSignerPrefix = "k8s.io/"
	CertProviderNone                   = "none"
	CertProviderCustom                 = "custom"
	CACertNamespaceConfigMapDataName   = "root-cert.pem"

	PodInfoAnnotationsPath = "./etc/dubbo/pod/annotations"

	StatPrefixDelimiter = ";"

	SailWellKnownDNSCertPath   = "./var/run/secrets/dubbod/tls/"
	SailWellKnownDNSCaCertPath = "./var/run/secrets/dubbod/ca/"

	DefaultSailTLSCert                = SailWellKnownDNSCertPath + "tls.crt"
	DefaultSailTLSKey                 = SailWellKnownDNSCertPath + "tls.key"
	DefaultSailTLSCaCert              = SailWellKnownDNSCaCertPath + "root-cert.pem"
	DefaultSailTLSCaCertAlternatePath = SailWellKnownDNSCertPath + "ca.crt"

	AlwaysReject = "internal.dubbo.io/webhook-always-reject"

	KubeSystemNamespace string = "kube-system"

	ThirdPartyJwtPath = "./var/run/secrets/tokens/istio-token"
)
