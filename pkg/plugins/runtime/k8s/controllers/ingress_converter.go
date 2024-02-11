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
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	kube_core "k8s.io/api/core/v1"
)

var NodePortAddressPriority = []kube_core.NodeAddressType{
	kube_core.NodeExternalIP,
	kube_core.NodeInternalIP,
}

func (p *PodConverter) IngressFor(
	ctx context.Context, zoneIngress *mesh_proto.ZoneIngress, pod *kube_core.Pod, service []*kube_core.Service,
) error {
	return nil
}
