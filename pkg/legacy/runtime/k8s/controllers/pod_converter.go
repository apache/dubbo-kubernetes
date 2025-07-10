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

	kube_core "k8s.io/api/core/v1"
	kube_client "sigs.k8s.io/controller-runtime/pkg/client"

	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	k8s_common "github.com/apache/dubbo-kubernetes/pkg/plugins/common/k8s"
	mesh_k8s "github.com/apache/dubbo-kubernetes/pkg/plugins/resources/k8s/native/api/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/runtime/k8s/metadata"
	util_k8s "github.com/apache/dubbo-kubernetes/pkg/plugins/runtime/k8s/util"
)

var converterLog = core.Log.WithName("discoveryengine").WithName("k8s").WithName("pod-to-dataplane-converter")

type PodConverter struct {
	ServiceGetter     kube_client.Reader
	NodeGetter        kube_client.Reader
	ResourceConverter k8s_common.Converter
	InboundConverter  InboundConverter
	Zone              string
}

func (p *PodConverter) PodToDataplane(
	ctx context.Context,
	dataplane *mesh_k8s.Dataplane,
	pod *kube_core.Pod,
	ns *kube_core.Namespace,
	services []*kube_core.Service,
	others []*mesh_k8s.Dataplane,
) error {
	dataplane.Mesh = util_k8s.MeshOfByAnnotation(pod, ns)
	dataplaneProto, err := p.dataplaneFor(ctx, pod, services, others)
	if err != nil {
		return err
	}
	dataplane.SetSpec(dataplaneProto)
	return nil
}

func (p *PodConverter) PodToIngress(ctx context.Context, zoneIngress *mesh_k8s.ZoneIngress, pod *kube_core.Pod, services []*kube_core.Service) error {
	logger := converterLog.WithValues("ZoneIngress.name", zoneIngress.Name, "Pod.name", pod.Name)
	// Start with the existing ZoneIngress spec so we won't override available services in Ingress section
	zoneIngressRes := core_mesh.NewZoneIngressResource()
	if err := p.ResourceConverter.ToCoreResource(zoneIngress, zoneIngressRes); err != nil {
		logger.Error(err, "unable to convert ZoneIngress k8s object into core resource")
		return err
	}

	if err := p.IngressFor(ctx, zoneIngressRes.Spec, pod, services); err != nil {
		return err
	}

	zoneIngress.SetSpec(zoneIngressRes.Spec)
	return nil
}

func (p *PodConverter) PodToEgress(ctx context.Context, zoneEgress *mesh_k8s.ZoneEgress, pod *kube_core.Pod, services []*kube_core.Service) error {
	logger := converterLog.WithValues("ZoneEgress.name", zoneEgress.Name, "Pod.name", pod.Name)
	// Start with the existing ZoneEgress spec
	zoneEgressRes := core_mesh.NewZoneEgressResource()
	if err := p.ResourceConverter.ToCoreResource(zoneEgress, zoneEgressRes); err != nil {
		logger.Error(err, "unable to convert ZoneEgress k8s object into core resource")
		return err
	}

	if err := p.EgressFor(ctx, zoneEgressRes.Spec, pod, services); err != nil {
		return err
	}

	zoneEgress.SetSpec(zoneEgressRes.Spec)
	return nil
}

func (p *PodConverter) dataplaneFor(
	ctx context.Context,
	pod *kube_core.Pod,
	services []*kube_core.Service,
	others []*mesh_k8s.Dataplane,
) (*mesh_proto.Dataplane, error) {
	dataplane := &mesh_proto.Dataplane{
		Networking: &mesh_proto.Dataplane_Networking{},
	}
	annotations := metadata.Annotations(pod.Annotations)

	dataplane.Networking.Address = pod.Status.PodIP
	ifaces, err := p.InboundConverter.InboundInterfacesFor(ctx, p.Zone, pod, services)
	if err != nil {
		return nil, err
	}
	dataplane.Networking.Inbound = ifaces

	ofaces, err := p.OutboundInterfacesFor(ctx, pod, others, []string{})
	if err != nil {
		return nil, err
	}
	dataplane.Networking.Outbound = ofaces

	probes, err := ProbesFor(pod)
	if err != nil {
		return nil, err
	}
	dataplane.Probes = probes

	adminPort, exist, err := annotations.GetUint32(metadata.DubboEnvoyAdminPort)
	if err != nil {
		return nil, err
	}
	if exist {
		dataplane.Networking.Admin = &mesh_proto.EnvoyAdmin{Port: adminPort}
	}

	return dataplane, nil
}
