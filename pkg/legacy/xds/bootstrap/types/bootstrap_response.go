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

type BootstrapVersion string

const (
	BootstrapV3 BootstrapVersion = "3"
)

// Bootstrap is sent to a client (Dubbo DP) by putting YAML into a response body.
// This YAML has no information about Bootstrap version therefore we put extra header with a version
// Value of this header is then used in CLI arg --bootstrap-version when Envoy is run
const BootstrapVersionHeader = "dubbo-bootstrap-version"

type BootstrapResponse struct {
	Bootstrap                 []byte                    `json:"bootstrap"`
	DubboSidecarConfiguration DubboSidecarConfiguration `json:"dubboSidecarConfiguration"`
}

type DubboSidecarConfiguration struct {
	Networking NetworkingConfiguration `json:"networking"`
	Metrics    MetricsConfiguration    `json:"metrics"`
}

type NetworkingConfiguration struct {
	IsUsingTransparentProxy bool   `json:"isUsingTransparentProxy"`
	CorefileTemplate        []byte `json:"corefileTemplate"`
	Address                 string `json:"address"`
}

type MetricsConfiguration struct {
	Aggregate []Aggregate `json:"aggregate"`
}

type Aggregate struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Port    uint32 `json:"port"`
	Path    string `json:"path"`
}
