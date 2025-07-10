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

package ingress

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

import (
	"golang.org/x/exp/slices"

	"google.golang.org/protobuf/proto"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	envoy "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/tags"
)

// tagSets represent map from tags (encoded as string) to number of instances
type tagSets map[serviceKey]uint32

type serviceKey struct {
	mesh string
	tags string
}

type serviceKeySlice []serviceKey

func (s serviceKeySlice) Len() int { return len(s) }
func (s serviceKeySlice) Less(i, j int) bool {
	return s[i].mesh < s[j].mesh || (s[i].mesh == s[j].mesh && s[i].tags < s[j].tags)
}
func (s serviceKeySlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (sk *serviceKey) String() string {
	return fmt.Sprintf("%s.%s", sk.tags, sk.mesh)
}

func (s tagSets) addInstanceOfTags(mesh string, tags envoy.Tags) {
	strTags := tags.String()
	s[serviceKey{tags: strTags, mesh: mesh}]++
}

func (s tagSets) toAvailableServices() []*mesh_proto.ZoneIngress_AvailableService {
	var result []*mesh_proto.ZoneIngress_AvailableService

	var keys []serviceKey
	for key := range s {
		keys = append(keys, key)
	}
	sort.Sort(serviceKeySlice(keys))

	for _, key := range keys {
		tags, _ := envoy.TagsFromString(key.tags) // ignore error since we control how string looks like
		result = append(result, &mesh_proto.ZoneIngress_AvailableService{
			Tags:      tags,
			Instances: s[key],
			Mesh:      key.mesh,
		})
	}
	return result
}

func UpdateAvailableServices(
	ctx context.Context,
	rm manager.ResourceManager,
	ingress *core_mesh.ZoneIngressResource,
	otherDataplanes []*core_mesh.DataplaneResource,
	tagFilters []string,
) error {
	availableServices := GetIngressAvailableServices(otherDataplanes, tagFilters)

	if availableServicesEqual(availableServices, ingress.Spec.GetAvailableServices()) {
		return nil
	}
	ingress.Spec.AvailableServices = availableServices
	if err := rm.Update(ctx, ingress); err != nil {
		return err
	}
	return nil
}

func availableServicesEqual(services []*mesh_proto.ZoneIngress_AvailableService, other []*mesh_proto.ZoneIngress_AvailableService) bool {
	if len(services) != len(other) {
		return false
	}
	for i := range services {
		if !proto.Equal(services[i], other[i]) {
			return false
		}
	}
	return true
}

func GetIngressAvailableServices(others []*core_mesh.DataplaneResource, tagFilters []string) []*mesh_proto.ZoneIngress_AvailableService {
	tagSets := tagSets{}
	for _, dp := range others {
		for _, dpInbound := range dp.Spec.GetNetworking().GetHealthyInbounds() {
			tags := map[string]string{}
			for key, value := range dpInbound.Tags {
				hasPrefix := func(tagFilter string) bool {
					return strings.HasPrefix(key, tagFilter)
				}
				if len(tagFilters) == 0 || slices.ContainsFunc(tagFilters, hasPrefix) {
					tags[key] = value
				}
			}
			tagSets.addInstanceOfTags(dp.GetMeta().GetMesh(), tags)
		}
	}
	return tagSets.toAvailableServices()
}
