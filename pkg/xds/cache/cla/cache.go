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

package cla

import (
	"context"
	"fmt"
	"time"
)

import (
	"google.golang.org/protobuf/proto"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/xds"
	"github.com/apache/dubbo-kubernetes/pkg/xds/cache/once"
	"github.com/apache/dubbo-kubernetes/pkg/xds/cache/sha256"
	envoy_common "github.com/apache/dubbo-kubernetes/pkg/xds/envoy"
	envoy_endpoints "github.com/apache/dubbo-kubernetes/pkg/xds/envoy/endpoints"
)

type Cache struct {
	cache *once.Cache
}

func NewCache(
	expirationTime time.Duration,
) (*Cache, error) {
	c, err := once.New(expirationTime, "cla_cache")
	if err != nil {
		return nil, err
	}
	return &Cache{
		cache: c,
	}, nil
}

func (c *Cache) GetCLA(ctx context.Context, meshName, meshHash string, cluster envoy_common.Cluster, apiVersion xds.APIVersion, endpointMap xds.EndpointMap) (proto.Message, error) {
	key := sha256.Hash(fmt.Sprintf("%s:%s:%s:%s", apiVersion, meshName, cluster.Hash(), meshHash))

	elt, err := c.cache.GetOrRetrieve(ctx, key, once.RetrieverFunc(func(ctx context.Context, key string) (interface{}, error) {
		matchTags := map[string]string{}
		for tag, val := range cluster.Tags() {
			if tag != mesh_proto.ServiceTag {
				matchTags[tag] = val
			}
		}

		endpoints := endpointMap[cluster.Service()]
		if len(matchTags) > 0 {
			endpoints = []xds.Endpoint{}
			for _, endpoint := range endpointMap[cluster.Service()] {
				if endpoint.ContainsTags(matchTags) {
					endpoints = append(endpoints, endpoint)
				}
			}
		}
		return envoy_endpoints.CreateClusterLoadAssignment(cluster.Name(), endpoints, apiVersion)
	}))
	if err != nil {
		return nil, err
	}
	return elt.(proto.Message), nil
}
