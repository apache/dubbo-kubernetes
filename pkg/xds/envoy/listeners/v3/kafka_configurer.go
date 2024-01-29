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

package v3

import (
	envoy_kafka "github.com/envoyproxy/go-control-plane/contrib/envoy/extensions/filters/network/kafka_broker/v3"

	"github.com/apache/dubbo-kubernetes/pkg/util/proto"
	util_xds "github.com/apache/dubbo-kubernetes/pkg/util/xds"
	envoy_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
)

type KafkaConfigurer struct {
	StatsName string
}

var _ FilterChainConfigurer = &KafkaConfigurer{}

func (c *KafkaConfigurer) Configure(filterChain *envoy_listener.FilterChain) error {
	pbst, err := proto.MarshalAnyDeterministic(
		&envoy_kafka.KafkaBroker{
			StatPrefix: util_xds.SanitizeMetric(c.StatsName),
		})
	if err != nil {
		return err
	}

	filterChain.Filters = append([]*envoy_listener.Filter{
		{
			Name: "envoy.filters.network.kafka_broker",
			ConfigType: &envoy_listener.Filter_TypedConfig{
				TypedConfig: pbst,
			},
		},
	}, filterChain.Filters...)
	return nil
}
