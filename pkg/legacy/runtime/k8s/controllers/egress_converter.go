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

	"github.com/pkg/errors"
	kube_core "k8s.io/api/core/v1"

	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/runtime/k8s/metadata"
)

func (p *PodConverter) EgressFor(
	ctx context.Context,
	zoneEgress *mesh_proto.ZoneEgress,
	pod *kube_core.Pod,
	services []*kube_core.Service,
) error {
	if len(services) != 1 {
		return errors.Errorf("egress should be matched by exactly one service. Matched %d services", len(services))
	}
	ifaces, err := p.InboundConverter.InboundInterfacesFor(ctx, p.Zone, pod, services)
	if err != nil {
		return errors.Wrap(err, "could not generate inbound interfaces")
	}
	if len(ifaces) != 1 {
		return errors.Errorf("generated %d inbound interfaces, expected 1. Interfaces: %v", len(ifaces), ifaces)
	}

	if zoneEgress.Networking == nil {
		zoneEgress.Networking = &mesh_proto.ZoneEgress_Networking{}
	}

	zoneEgress.Zone = p.Zone
	zoneEgress.Networking.Address = pod.Status.PodIP
	zoneEgress.Networking.Port = ifaces[0].Port

	adminPort, exist, err := metadata.Annotations(pod.Annotations).GetUint32(metadata.DubboEnvoyAdminPort)
	if err != nil {
		return err
	}
	if exist {
		zoneEgress.Networking.Admin = &mesh_proto.EnvoyAdmin{Port: adminPort}
	}

	return nil
}
