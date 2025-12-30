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

package controller

import (
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/model"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/serviceregistry/kube"
	labelutil "github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/serviceregistry/util/label"
	"github.com/apache/dubbo-kubernetes/pkg/config/labels"
	"github.com/apache/dubbo-kubernetes/pkg/network"
	v1 "k8s.io/api/core/v1"
)

type EndpointBuilder struct {
	controller     controllerInterface
	labels         labels.Instance
	metaNetwork    network.ID
	serviceAccount string
	workloadName   string
	namespace      string
	hostname       string
	subDomain      string
	nodeName       string
}

func (c *Controller) NewEndpointBuilder(pod *v1.Pod) *EndpointBuilder {
	var sa, namespace, hostname, subdomain, ip, node string
	var podLabels labels.Instance
	if pod != nil {
		sa = kube.SecureNamingSAN(pod, c.meshWatcher.Mesh())
		podLabels = pod.Labels
		namespace = pod.Namespace
		subdomain = pod.Spec.Subdomain
		if subdomain != "" {
			hostname = pod.Spec.Hostname
			if hostname == "" {
				hostname = pod.Name
			}
		}
		node = pod.Spec.NodeName
	}
	out := &EndpointBuilder{
		controller:     c,
		serviceAccount: sa,
		namespace:      namespace,
		hostname:       hostname,
		subDomain:      subdomain,
		labels:         podLabels,
		nodeName:       node,
	}
	networkID := out.endpointNetwork(ip)
	out.labels = labelutil.AugmentLabels(podLabels, c.Cluster(), "", node, networkID)
	return out
}

func (b *EndpointBuilder) buildDubboEndpoint(
	endpointAddress string,
	endpointPort int32,
	svcPortName string,
	discoverabilityPolicy model.EndpointDiscoverabilityPolicy,
	healthStatus model.HealthStatus,
	sendUnhealthy bool,
) *model.DubboEndpoint {
	if b == nil {
		return nil
	}

	// in case pod is not found when init EndpointBuilder.
	networkID := network.ID(b.labels["topology.dubbo.apache.org/network"])
	if networkID == "" {
		networkID = b.endpointNetwork(endpointAddress)
		b.labels["topology.dubbo.apache.org/network"] = string(networkID)
	}

	return &model.DubboEndpoint{
		Labels:                 b.labels,
		ServiceAccount:         b.serviceAccount,
		Addresses:              []string{endpointAddress},
		EndpointPort:           uint32(endpointPort),
		ServicePortName:        svcPortName,
		Network:                networkID,
		Namespace:              b.namespace,
		HostName:               b.hostname,
		SubDomain:              b.subDomain,
		DiscoverabilityPolicy:  discoverabilityPolicy,
		HealthStatus:           healthStatus,
		SendUnhealthyEndpoints: sendUnhealthy,
		NodeName:               b.nodeName,
	}
}

func (b *EndpointBuilder) endpointNetwork(endpointIP string) network.ID {
	// If we're building the endpoint based on proxy meta, prefer the injected ISTIO_META_NETWORK value.
	if b.metaNetwork != "" {
		return b.metaNetwork
	}

	return b.controller.Network(endpointIP, b.labels)
}
