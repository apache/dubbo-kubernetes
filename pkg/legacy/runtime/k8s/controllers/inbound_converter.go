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

package controllers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	kube_core "k8s.io/api/core/v1"
	kube_client "sigs.k8s.io/controller-runtime/pkg/client"

	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	util_k8s "github.com/apache/dubbo-kubernetes/pkg/plugins/runtime/k8s/util"
)

type InboundConverter struct {
	NameExtractor NameExtractor
}

func inboundForService(zone string, pod *kube_core.Pod, service *kube_core.Service) []*mesh_proto.Dataplane_Networking_Inbound {
	var ifaces []*mesh_proto.Dataplane_Networking_Inbound
	for i := range service.Spec.Ports {
		svcPort := service.Spec.Ports[i]
		if svcPort.Protocol != "" && svcPort.Protocol != kube_core.ProtocolTCP {
			// ignore non-TCP ports
			continue
		}
		containerPort, container, err := util_k8s.FindPort(pod, &svcPort)
		if err != nil {
			converterLog.Error(err, "failed to find a container port in a given Pod that would match a given Service port", "namespace", pod.Namespace, "podName", pod.Name, "serviceName", service.Name, "servicePortName", svcPort.Name)
			// ignore those cases where a Pod doesn't have all the ports a Service has
			continue
		}

		tags := InboundTagsForService(zone, pod, service, &svcPort)
		state := mesh_proto.Dataplane_Networking_Inbound_Ready
		health := mesh_proto.Dataplane_Networking_Inbound_Health{
			Ready: true,
		}

		// if container is not equal nil then port is explicitly defined as containerPort so we're able
		// to figure out which container implements which service. Since we know container we can check its status
		// and map it to the Dataplane health
		if container != nil {
			if cs := util_k8s.FindContainerStatus(pod, container.Name); cs != nil && !cs.Ready {
				state = mesh_proto.Dataplane_Networking_Inbound_NotReady
				health.Ready = false
			}
		}

		ifaces = append(ifaces, &mesh_proto.Dataplane_Networking_Inbound{
			Port:   uint32(containerPort),
			Tags:   tags,
			State:  state,
			Health: &health, // write health for backwards compatibility with Dubbo
		})
	}

	return ifaces
}

func inboundForServiceless(zone string, pod *kube_core.Pod, name string) *mesh_proto.Dataplane_Networking_Inbound {
	// The Pod does not have any services associated with it, just get the data from the Pod itself

	// We still need that extra listener with a service because it is required in many places of the code (e.g. mTLS)
	// TCPPortReserved, is a special port that will never be allocated from the TCP/IP stack. We use it as special
	// designator that this is actually a service-less inbound.

	// NOTE: It is cleaner to implement an equivalent of Gateway which is inbound-less dataplane. However such approch
	// will create lots of code changes to account for this other type of dataplne (we already have GW and Ingress),
	// including GUI and CLI changes

	tags := InboundTagsForPod(zone, pod, name)
	state := mesh_proto.Dataplane_Networking_Inbound_Ready
	health := mesh_proto.Dataplane_Networking_Inbound_Health{
		Ready: true,
	}

	return &mesh_proto.Dataplane_Networking_Inbound{
		Port:   mesh_proto.TCPPortReserved,
		Tags:   tags,
		State:  state,
		Health: &health, // write health for backwards compatibility with Dubbo
	}
}

func (i *InboundConverter) InboundInterfacesFor(ctx context.Context, zone string, pod *kube_core.Pod, services []*kube_core.Service) ([]*mesh_proto.Dataplane_Networking_Inbound, error) {
	var ifaces []*mesh_proto.Dataplane_Networking_Inbound
	for _, svc := range services {
		// Services of ExternalName type should not have any selectors.
		// Kubernetes does not validate this, so in rare cases, a service of
		// ExternalName type could point to a workload inside the mesh. If this
		// happens, we would incorrectly generate inbounds including
		// ExternalName service. We do not currently support ExternalName
		// services, so we can safely skip them from processing.
		if svc.Spec.Type != kube_core.ServiceTypeExternalName {
			ifaces = append(ifaces, inboundForService(zone, pod, svc)...)
		}
	}

	if len(ifaces) == 0 {
		if len(services) > 0 {
			return nil, errors.Errorf("A service that selects pod %s was found, but it doesn't match any container ports.", pod.GetName())
		}
		name, _, err := i.NameExtractor.Name(ctx, pod)
		if err != nil {
			return nil, err
		}

		ifaces = append(ifaces, inboundForServiceless(zone, pod, name))
	}
	return ifaces, nil
}

func InboundTagsForService(zone string, pod *kube_core.Pod, svc *kube_core.Service, svcPort *kube_core.ServicePort) map[string]string {
	logger := converterLog.WithValues("pod", pod.Name, "namespace", pod.Namespace)
	tags := map[string]string{}
	var ignoredLabels []string
	for key, value := range pod.Labels {
		if value == "" {
			continue
		}
		if strings.Contains(key, "dubbo.io/") {
			ignoredLabels = append(ignoredLabels, key)
			continue
		}
		tags[key] = value
	}
	if len(ignoredLabels) > 0 {
		logger.Info("ignoring internal labels when converting labels to tags", "label", strings.Join(ignoredLabels, ","))
	}
	tags[mesh_proto.KubeNamespaceTag] = pod.Namespace
	tags[mesh_proto.KubeServiceTag] = svc.Name
	tags[mesh_proto.KubePortTag] = strconv.Itoa(int(svcPort.Port))
	tags[mesh_proto.ServiceTag] = util_k8s.ServiceTag(kube_client.ObjectKeyFromObject(svc), &svcPort.Port)
	if zone != "" {
		tags[mesh_proto.ZoneTag] = zone
	}
	// For provided gateway we should ignore the protocol tag
	protocol := ProtocolTagFor(svc, svcPort)

	tags[mesh_proto.ProtocolTag] = protocol

	if isHeadlessService(svc) {
		tags[mesh_proto.InstanceTag] = pod.Name
	}
	return tags
}

// ProtocolTagFor infers service protocol from a `<port>.service.dubbo.io/protocol` annotation or `appProtocol`.
func ProtocolTagFor(svc *kube_core.Service, svcPort *kube_core.ServicePort) string {
	var protocolValue string
	protocolAnnotation := fmt.Sprintf("%d.service.dubbo.io/protocol", svcPort.Port)

	if svcPort.AppProtocol != nil {
		protocolValue = *svcPort.AppProtocol
		// `appProtocol` can be any protocol and if we don't explicitly support
		// it, let the default below take effect
		if core_mesh.ParseProtocol(protocolValue) == core_mesh.ProtocolUnknown {
			protocolValue = ""
		}
	}

	if explicitDubboProtocol, ok := svc.Annotations[protocolAnnotation]; ok && protocolValue == "" {
		protocolValue = explicitDubboProtocol
	}

	if protocolValue == "" {
		// if `appProtocol` or `<port>.service.dubbo.io/protocol` is missing or has an empty value
		// we want Dataplane to have a `protocol: tcp` tag in order to get user's attention
		protocolValue = core_mesh.ProtocolTCP
	}

	// if `<port>.service.dubbo.io/protocol` field is present but has an invalid value
	// we still want Dataplane to have a `protocol: <lowercase value>` tag in order to make it clear
	// to a user that at least `<port>.service.dubbo.io/protocol` has an effect
	return strings.ToLower(protocolValue)
}

func InboundTagsForPod(zone string, pod *kube_core.Pod, name string) map[string]string {
	tags := util_k8s.CopyStringMap(pod.Labels)
	for key, value := range tags {
		if value == "" {
			delete(tags, key)
		}
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	tags[mesh_proto.KubeNamespaceTag] = pod.Namespace
	tags[mesh_proto.ServiceTag] = fmt.Sprintf("%s_%s_svc", name, pod.Namespace)
	if zone != "" {
		tags[mesh_proto.ZoneTag] = zone
	}
	tags[mesh_proto.ProtocolTag] = core_mesh.ProtocolTCP
	tags[mesh_proto.InstanceTag] = pod.Name

	return tags
}
