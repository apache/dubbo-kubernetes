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

package kube

import (
	"strings"

	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/kube"
	"github.com/apache/dubbo-kubernetes/pkg/config/visibility"
	"github.com/apache/dubbo-kubernetes/pkg/spiffe"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"github.com/apache/dubbo-kubernetes/sail/pkg/serviceregistry/provider"
	"istio.io/api/annotation"
	meshconfig "istio.io/api/mesh/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func ServiceHostname(name, namespace, domainSuffix string) host.Name {
	return host.Name(name + "." + namespace + "." + "svc" + "." + domainSuffix) // Format: "%s.%s.svc.%s"
}

func ConvertService(svc corev1.Service, domainSuffix string, clusterID cluster.ID, mesh *meshconfig.MeshConfig) *model.Service {
	addrs := []string{constants.UnspecifiedIP}
	resolution := model.ClientSideLB
	externalName := ""
	nodeLocal := false

	if svc.Spec.Type == corev1.ServiceTypeExternalName && svc.Spec.ExternalName != "" {
		externalName = svc.Spec.ExternalName
		resolution = model.Alias
	}

	if svc.Spec.InternalTrafficPolicy != nil && *svc.Spec.InternalTrafficPolicy == corev1.ServiceInternalTrafficPolicyLocal {
		nodeLocal = true
	}

	if svc.Spec.ClusterIP == corev1.ClusterIPNone { // headless services should not be load balanced
		resolution = model.Passthrough
	} else if svc.Spec.ClusterIP != "" {
		addrs[0] = svc.Spec.ClusterIP
		if len(svc.Spec.ClusterIPs) > 1 {
			addrs = svc.Spec.ClusterIPs
		}
	}

	ports := make([]*model.Port, 0, len(svc.Spec.Ports))
	for _, port := range svc.Spec.Ports {
		ports = append(ports, convertPort(port))
	}
	var exportTo sets.Set[visibility.Instance]
	serviceaccounts := make([]string, 0)
	if svc.Annotations[annotation.AlphaCanonicalServiceAccounts.Name] != "" {
		serviceaccounts = append(serviceaccounts, strings.Split(svc.Annotations[annotation.AlphaCanonicalServiceAccounts.Name], ",")...)
	}
	if svc.Annotations[annotation.AlphaKubernetesServiceAccounts.Name] != "" {
		for _, ksa := range strings.Split(svc.Annotations[annotation.AlphaKubernetesServiceAccounts.Name], ",") {
			serviceaccounts = append(serviceaccounts, kubeToDubboServiceAccount(ksa, svc.Namespace, mesh))
		}
	}
	if svc.Annotations[annotation.NetworkingExportTo.Name] != "" {
		namespaces := strings.Split(svc.Annotations[annotation.NetworkingExportTo.Name], ",")
		exportTo = sets.NewWithLength[visibility.Instance](len(namespaces))
		for _, ns := range namespaces {
			ns = strings.TrimSpace(ns)
			exportTo.Insert(visibility.Instance(ns))
		}
	}

	dubboService := &model.Service{
		Hostname: ServiceHostname(svc.Name, svc.Namespace, domainSuffix),
		ClusterVIPs: model.AddressMap{
			Addresses: map[cluster.ID][]string{
				clusterID: addrs,
			},
		},
		Ports:           ports,
		DefaultAddress:  addrs[0],
		Resolution:      resolution,
		ServiceAccounts: serviceaccounts,
		ResourceVersion: svc.ResourceVersion,
		Attributes: model.ServiceAttributes{
			ServiceRegistry: provider.Kubernetes,
			Name:            svc.Name,
			Namespace:       svc.Namespace,
			Labels:          svc.Labels,
			ExportTo:        exportTo,
			LabelSelectors:  svc.Spec.Selector,
		},
	}

	switch svc.Spec.Type {
	case corev1.ServiceTypeNodePort:
		if _, ok := svc.Annotations[annotation.TrafficNodeSelector.Name]; !ok {
			break
		}
		// store the service port to node port mappings
		portMap := make(map[uint32]uint32)
		for _, p := range svc.Spec.Ports {
			portMap[uint32(p.Port)] = uint32(p.NodePort)
		}
		dubboService.Attributes.ClusterExternalPorts = map[cluster.ID]map[uint32]uint32{clusterID: portMap}
		// address mappings will be done elsewhere
	case corev1.ServiceTypeLoadBalancer:
		if len(svc.Status.LoadBalancer.Ingress) > 0 {
			var lbAddrs []string
			for _, ingress := range svc.Status.LoadBalancer.Ingress {
				if len(ingress.IP) > 0 {
					lbAddrs = append(lbAddrs, ingress.IP)
				} else if len(ingress.Hostname) > 0 {
					// DO NOT resolve the DNS here. In environments like AWS, the ELB hostname
					// does not have a repeatable DNS address and IPs resolved at an earlier point
					// in time may not work. So, when we get just hostnames instead of IPs, we need
					// to smartly switch from EDS to strict_dns rather than doing the naive thing of
					// resolving the DNS name and hoping the resolution is one-time task.
					lbAddrs = append(lbAddrs, ingress.Hostname)
				}
			}
			if len(lbAddrs) > 0 {
				if dubboService.Attributes.ClusterExternalAddresses == nil {
					dubboService.Attributes.ClusterExternalAddresses = &model.AddressMap{}
				}
				dubboService.Attributes.ClusterExternalAddresses.SetAddressesFor(clusterID, lbAddrs)
			}
		}
	}

	dubboService.Attributes.Type = string(svc.Spec.Type)
	dubboService.Attributes.ExternalName = externalName
	dubboService.Attributes.TrafficDistribution = model.GetTrafficDistribution(svc.Spec.TrafficDistribution, svc.Annotations)
	dubboService.Attributes.NodeLocal = nodeLocal
	dubboService.Attributes.PublishNotReadyAddresses = svc.Spec.PublishNotReadyAddresses
	if len(svc.Spec.ExternalIPs) > 0 {
		if dubboService.Attributes.ClusterExternalAddresses == nil {
			dubboService.Attributes.ClusterExternalAddresses = &model.AddressMap{}
		}
		dubboService.Attributes.ClusterExternalAddresses.AddAddressesFor(clusterID, svc.Spec.ExternalIPs)
	}
	return dubboService
}

func convertPort(port corev1.ServicePort) *model.Port {
	return &model.Port{
		Name:     port.Name,
		Port:     int(port.Port),
		Protocol: kube.ConvertProtocol(port.Port, port.Name, port.Protocol, port.AppProtocol),
	}
}

func kubeToDubboServiceAccount(saname string, ns string, mesh *meshconfig.MeshConfig) string {
	return spiffe.MustGenSpiffeURI(mesh, ns, saname)
}

func SecureNamingSAN(pod *corev1.Pod, mesh *meshconfig.MeshConfig) string {
	return spiffe.MustGenSpiffeURI(mesh, pod.Namespace, pod.Spec.ServiceAccountName)
}
