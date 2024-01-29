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

package clusters

import (
	util_proto "github.com/apache/dubbo-kubernetes/pkg/util/proto"
	envoy_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_upstream_http "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"
	"google.golang.org/protobuf/types/known/anypb"
)

func UpdateCommonHttpProtocolOptions(cluster *envoy_cluster.Cluster, fn func(*envoy_upstream_http.HttpProtocolOptions)) error {
	if cluster.TypedExtensionProtocolOptions == nil {
		cluster.TypedExtensionProtocolOptions = map[string]*anypb.Any{}
	}
	options := &envoy_upstream_http.HttpProtocolOptions{}
	if any := cluster.TypedExtensionProtocolOptions["envoy.extensions.upstreams.http.v3.HttpProtocolOptions"]; any != nil {
		if err := util_proto.UnmarshalAnyTo(any, options); err != nil {
			return err
		}
	}

	fn(options)

	pbst, err := util_proto.MarshalAnyDeterministic(options)
	if err != nil {
		return err
	}
	cluster.TypedExtensionProtocolOptions["envoy.extensions.upstreams.http.v3.HttpProtocolOptions"] = pbst
	return nil
}
