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

package grpcgen

import (
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	v1 "github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/xds/v1"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	tlsv1 "github.com/kdubbo/xds-api/extensions/transport_sockets/tls/v1"
)

var log = dubbolog.RegisterScope("grpcgen", "xDS Generator for Proxyless gRPC")

type GrpcConfigGenerator struct{}

var _ model.XdsDeltaResourceGenerator = &GrpcConfigGenerator{}

func (g *GrpcConfigGenerator) Generate(proxy *model.Proxy, w *model.WatchedResource, req *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	// Extract requested resource names from WatchedResource
	// If ResourceNames is empty (wildcard request), pass empty slice
	// BuildListeners will handle empty names as wildcard request and generate all listeners
	var requestedNames []string
	if w != nil && w.ResourceNames != nil && len(w.ResourceNames) > 0 {
		requestedNames = w.ResourceNames.UnsortedList()
	}

	switch w.TypeUrl {
	case v1.ListenerType:
		// Pass requested names to BuildListeners to ensure consistent behavior
		// When requestedNames is empty (wildcard), BuildListeners generates all listeners
		// When requestedNames is non-empty, BuildListeners only generates requested listeners
		return g.BuildListeners(proxy, req.Push, requestedNames), model.DefaultXdsLogDetails, nil
	case v1.ClusterType:
		return g.BuildClusters(proxy, req.Push, requestedNames), model.DefaultXdsLogDetails, nil
	case v1.RouteType:
		resources, logDetails := g.BuildHTTPRoutes(proxy, req, requestedNames)
		return resources, logDetails, nil
	}

	return nil, model.DefaultXdsLogDetails, nil
}

func (g *GrpcConfigGenerator) GenerateDeltas(proxy *model.Proxy, req *model.PushRequest, w *model.WatchedResource) (model.Resources, model.DeletedResources, model.XdsLogDetails, bool, error) {
	if w == nil {
		return nil, nil, model.DefaultXdsLogDetails, false, nil
	}
	if w.TypeUrl != v1.ClusterType || !grpcCanUseDeltaClusters(req) {
		resources, logDetails, err := g.Generate(proxy, w, req)
		return resources, nil, logDetails, false, err
	}

	changedNames := grpcDeltaClusterNames(w, req)
	if len(changedNames) == 0 {
		return nil, nil, model.DefaultXdsLogDetails, true, nil
	}
	resources := g.BuildClusters(proxy, req.Push, sets.SortedList(changedNames))

	built := sets.NewWithLength[string](len(resources))
	for _, resource := range resources {
		if resource != nil {
			built.Insert(resource.Name)
		}
	}
	removed := changedNames.Copy()
	for name := range built {
		removed.Delete(name)
	}

	logDetails := model.DefaultXdsLogDetails
	logDetails.Incremental = true
	return resources, sets.SortedList(removed), logDetails, true, nil
}

func grpcCanUseDeltaClusters(req *model.PushRequest) bool {
	if req == nil || req.Forced || len(req.ConfigsUpdated) == 0 {
		return false
	}
	for cfg := range req.ConfigsUpdated {
		if cfg.Kind != kind.Service || cfg.Name == "" {
			return false
		}
	}
	return true
}

func grpcDeltaClusterNames(w *model.WatchedResource, req *model.PushRequest) sets.String {
	updatedServices := sets.New[string]()
	for cfg := range req.ConfigsUpdated {
		updatedServices.Insert(cfg.Name)
	}
	changed := sets.New[string]()
	for clusterName := range w.ResourceNames {
		if updatedServices.Contains(model.ParseSubsetKeyHostname(clusterName)) {
			changed.Insert(clusterName)
		}
	}
	return changed
}

// buildCommonTLSContext creates a TLS context that matches gRPC xDS expectations.
//   - Uses certificate provider "default" for workload certs and root CA
//   - sans, when non-empty, restricts accepted peer certificates to the given
//     SPIFFE identities. Outbound clusters pass the target service's identities so
//     a workload with a valid mesh certificate cannot impersonate another service.
//     Inbound listeners pass nil: any authenticated mesh identity is accepted at
//     the transport layer and fine-grained restrictions belong to AuthorizationPolicy.
func buildCommonTLSContext(sans []string) *tlsv1.CommonTlsContext {
	return &tlsv1.CommonTlsContext{
		// Workload certificate provider instance (SPIFFE workload cert chain)
		TlsCertificateCertificateProviderInstance: &tlsv1.CommonTlsContext_CertificateProviderInstance{
			InstanceName:    "default",
			CertificateName: "default",
		},
		// Root CA provider instance
		ValidationContextType: &tlsv1.CommonTlsContext_CombinedValidationContext{
			CombinedValidationContext: &tlsv1.CommonTlsContext_CombinedCertificateValidationContext{
				ValidationContextCertificateProviderInstance: &tlsv1.CommonTlsContext_CertificateProviderInstance{
					InstanceName:    "default",
					CertificateName: "ROOTCA",
				},
				// The certificate provider instance (ROOTCA) provides the root CA
				// used for chain validation; match_subject_alt_names pins the peer
				// identity on top of that.
				DefaultValidationContext: &tlsv1.CertificateValidationContext{
					MatchSubjectAltNames: sans,
				},
			},
		},
	}
}
