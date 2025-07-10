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

package types

type BootstrapRequest struct {
	Mesh               string  `json:"mesh"`
	Name               string  `json:"name"`
	ProxyType          string  `json:"proxyType"`
	DataplaneToken     string  `json:"dataplaneToken,omitempty"`
	DataplaneTokenPath string  `json:"dataplaneTokenPath,omitempty"`
	DataplaneResource  string  `json:"dataplaneResource,omitempty"`
	Host               string  `json:"-"`
	Version            Version `json:"version"`
	// CaCert is a PEM-encoded CA cert that DP uses to verify CP
	CaCert              string            `json:"caCert"`
	DynamicMetadata     map[string]string `json:"dynamicMetadata"`
	DNSPort             uint32            `json:"dnsPort,omitempty"`
	EmptyDNSPort        uint32            `json:"emptyDnsPort,omitempty"`
	OperatingSystem     string            `json:"operatingSystem"`
	Features            []string          `json:"features"`
	Resources           ProxyResources    `json:"resources"`
	Workdir             string            `json:"workdir"`
	AccessLogSocketPath string            `json:"accessLogSocketPath"`
	MetricsResources    MetricsResources  `json:"metricsResources"`
}

type Version struct {
	DubboDp DubboDpVersion `json:"dubboDp"`
	Envoy   EnvoyVersion   `json:"envoy"`
}

type MetricsResources struct {
	SocketPath string `json:"socketPath"`
	CertPath   string `json:"certPath"`
	KeyPath    string `json:"keyPath"`
}

type DubboDpVersion struct {
	Version   string `json:"version"`
	GitTag    string `json:"gitTag"`
	GitCommit string `json:"gitCommit"`
	BuildDate string `json:"buildDate"`
}

type EnvoyVersion struct {
	Version           string `json:"version"`
	Build             string `json:"build"`
	DubboDpCompatible bool   `json:"dubboDpCompatible"`
}

// ProxyResources contains information about what resources this proxy has
// available
type ProxyResources struct {
	MaxHeapSizeBytes uint64 `json:"maxHeapSizeBytes"`
}
