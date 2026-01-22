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

package kube

import (
	meshv1alpha1 "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/serviceregistry/provider"
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/spiffe"
	corev1 "k8s.io/api/core/v1"
)

func ServiceHostname(name, namespace, domainSuffix string) host.Name {
	return host.Name(name + "." + namespace + "." + "svc" + "." + domainSuffix) // Format: "%s.%s.svc.%s"
}

func ConvertService(svc corev1.Service, domainSuffix string, clusterID cluster.ID, mesh *meshv1alpha1.MeshGlobalConfig) *model.Service {
	addrs := []string{constants.UnspecifiedIP}
	resolution := model.ClientSideLB
	externalName := ""

	if svc.Spec.Type == corev1.ServiceTypeExternalName && svc.Spec.ExternalName != "" {
		externalName = svc.Spec.ExternalName
		resolution = model.Alias
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
		ResourceVersion: svc.ResourceVersion,
		Attributes: model.ServiceAttributes{
			ServiceRegistry: provider.Kubernetes,
			Name:            svc.Name,
			Namespace:       svc.Namespace,
			Labels:          svc.Labels,
			LabelSelectors:  svc.Spec.Selector,
		},
	}

	dubboService.Attributes.Type = string(svc.Spec.Type)
	dubboService.Attributes.ExternalName = externalName
	dubboService.Attributes.PublishNotReadyAddresses = svc.Spec.PublishNotReadyAddresses
	return dubboService
}

func convertPort(port corev1.ServicePort) *model.Port {
	return &model.Port{
		Name: port.Name,
		Port: int(port.Port),
	}
}

func SecureNamingSAN(pod *corev1.Pod, mesh *meshv1alpha1.MeshGlobalConfig) string {
	return spiffe.MustGenSpiffeURI(mesh, pod.Namespace, pod.Spec.ServiceAccountName)
}
