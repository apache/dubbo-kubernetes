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

package topology

import (
	"net"
)

import (
	"github.com/pkg/errors"

	"google.golang.org/protobuf/proto"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/dns/lookup"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
)

// ResolveDataplaneAddress resolves 'dataplane.networking.address' if it has DNS name in it. This is a crucial feature for
// some environments specifically AWS ECS. Dataplane resource has to be created before running Dubbo DP, but IP address
// will be assigned only after container's start. Envoy EDS doesn't support DNS names, that's why Dubbo CP resolves
// addresses before sending resources to the proxy.
func ResolveDataplaneAddress(lookupIPFunc lookup.LookupIPFunc, dataplane *core_mesh.DataplaneResource) (*core_mesh.DataplaneResource, error) {
	if dataplane.Spec.Networking.Address == "" {
		return nil, errors.New("Dataplane address must always be set")
	}
	ip, err := lookupFirstIp(lookupIPFunc, dataplane.Spec.Networking.Address)
	if err != nil {
		return nil, err
	}
	aip, err := lookupFirstIp(lookupIPFunc, dataplane.Spec.Networking.AdvertisedAddress)
	if err != nil {
		return nil, err
	}
	if ip != "" || aip != "" { // only if we resolve any address, in most cases this is IP not a hostname
		dpSpec := proto.Clone(dataplane.Spec).(*mesh_proto.Dataplane)
		if ip != "" {
			dpSpec.Networking.Address = ip
		}
		if aip != "" {
			dpSpec.Networking.AdvertisedAddress = aip
		}
		return &core_mesh.DataplaneResource{
			Meta: dataplane.Meta,
			Spec: dpSpec,
		}, nil
	}
	return dataplane, nil
}

func lookupFirstIp(lookupIPFunc lookup.LookupIPFunc, address string) (string, error) {
	if address == "" || net.ParseIP(address) != nil { // There's either no address or it's already an ip so nothing to do
		return "", nil
	}
	// Resolve it!
	ips, err := lookupIPFunc(address)
	if err != nil {
		return "", err
	}
	if len(ips) == 0 {
		return "", errors.Errorf("can't resolve address %v", address)
	}
	// Pick the first lexicographic order ip (to make resolution deterministic
	minIp := ""
	for i := range ips {
		s := ips[i].String()
		if minIp == "" || s < minIp {
			minIp = s
		}
	}
	return minIp, nil
}
