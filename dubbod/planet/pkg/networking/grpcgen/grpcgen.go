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

package grpcgen

import (
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/model"
	v3 "github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/xds/v3"
	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
)

var log = dubbolog.RegisterScope("grpcgen", "xDS Generator for Proxyless gRPC")

type GrpcConfigGenerator struct{}

func (g *GrpcConfigGenerator) Generate(proxy *model.Proxy, w *model.WatchedResource, req *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	// Extract requested resource names from WatchedResource
	// If ResourceNames is empty (wildcard request), pass empty slice
	// BuildListeners will handle empty names as wildcard request and generate all listeners
	var requestedNames []string
	if w != nil && w.ResourceNames != nil && len(w.ResourceNames) > 0 {
		requestedNames = w.ResourceNames.UnsortedList()
	}

	switch w.TypeUrl {
	case v3.ListenerType:
		// Pass requested names to BuildListeners to ensure consistent behavior
		// When requestedNames is empty (wildcard), BuildListeners generates all listeners
		// When requestedNames is non-empty, BuildListeners only generates requested listeners
		return g.BuildListeners(proxy, req.Push, requestedNames), model.DefaultXdsLogDetails, nil
	case v3.ClusterType:
		return g.BuildClusters(proxy, req.Push, requestedNames), model.DefaultXdsLogDetails, nil
	case v3.RouteType:
		return g.BuildHTTPRoutes(proxy, req.Push, requestedNames), model.DefaultXdsLogDetails, nil
	}

	return nil, model.DefaultXdsLogDetails, nil
}

// buildCommonTLSContext creates a TLS context that matches gRPC xDS expectations.
// It is adapted from Istio's buildCommonTLSContext implementation, but kept minimal:
// - Uses certificate provider "default" for workload certs and root CA
// - Does not configure explicit SAN matches (left to future hardening)
func buildCommonTLSContext() *tlsv3.CommonTlsContext {
	return &tlsv3.CommonTlsContext{
		// Workload certificate provider instance (SPIFFE workload cert chain)
		TlsCertificateCertificateProviderInstance: &tlsv3.CommonTlsContext_CertificateProviderInstance{
			InstanceName:    "default",
			CertificateName: "default",
		},
		// Root CA provider instance
		ValidationContextType: &tlsv3.CommonTlsContext_CombinedValidationContext{
			CombinedValidationContext: &tlsv3.CommonTlsContext_CombinedCertificateValidationContext{
				ValidationContextCertificateProviderInstance: &tlsv3.CommonTlsContext_CertificateProviderInstance{
					InstanceName:    "default",
					CertificateName: "ROOTCA",
				},
				// DefaultValidationContext: Configure basic certificate validation
				// The certificate provider instance (ROOTCA) provides the root CA for validation
				// For gRPC proxyless, we rely on the certificate provider for root CA validation
				// SAN matching can be added later if needed for stricter validation
				DefaultValidationContext: &tlsv3.CertificateValidationContext{
					// Trust the root CA from the certificate provider
					// The certificate provider instance "default" with "ROOTCA" will provide
					// the root CA certificates for validating peer certificates
				},
			},
		},
	}
}
