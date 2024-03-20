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

package multizone

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
	config_types "github.com/apache/dubbo-kubernetes/pkg/config/types"
)

type DdsServerConfig struct {
	config.BaseConfig

	// Port of a gRPC server that serves Dubbo Discovery Service (DDS).
	GrpcPort uint32 `json:"grpcPort" envconfig:"dubbo_multizone_global_dds_grpc_port"`
	// Interval for refreshing state of the world
	RefreshInterval config_types.Duration `json:"refreshInterval" envconfig:"dubbo_multizone_global_dds_refresh_interval"`
}
