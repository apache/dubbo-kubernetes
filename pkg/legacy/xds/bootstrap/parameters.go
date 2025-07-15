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

package bootstrap

import (
	"time"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/xds/bootstrap/types"
)

type DubboDpBootstrap struct {
	AggregateMetricsConfig []AggregateMetricsConfig
	NetworkingConfig       NetworkingConfig
}
type NetworkingConfig struct {
	IsUsingTransparentProxy bool
	CorefileTemplate        []byte
	Address                 string
}

type AggregateMetricsConfig struct {
	Name    string
	Path    string
	Address string
	Port    uint32
}
type configParameters struct {
	Id                  string
	Service             string
	AdminAddress        string
	AdminPort           uint32
	AdminAccessLogPath  string
	XdsHost             string
	XdsPort             uint32
	XdsConnectTimeout   time.Duration
	Workdir             string
	AccessLogSocketPath string
	MetricsSocketPath   string
	MetricsCertPath     string
	MetricsKeyPath      string
	DataplaneToken      string
	DataplaneTokenPath  string
	DataplaneResource   string
	CertBytes           []byte
	Version             *mesh_proto.Version
	HdsEnabled          bool
	DynamicMetadata     map[string]string
	DNSPort             uint32
	EmptyDNSPort        uint32
	ProxyType           string
	Features            []string
	IsGatewayDataplane  bool
	Resources           types.ProxyResources
}
