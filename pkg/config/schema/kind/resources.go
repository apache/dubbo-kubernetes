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

package kind

const (
	Unknown Kind = iota
	Address
	CustomResourceDefinition
	MeshConfig
	MeshNetworks
	Namespace
	Pod
	Secret
	Service
	ServiceAccount
	ServiceEntry
	StatefulSet
	ValidatingWebhookConfiguration
	MutatingWebhookConfiguration
	PeerAuthentication
	ServiceRoute
	SubsetRule
)

func (k Kind) String() string {
	switch k {
	case Address:
		return "Address"
	case CustomResourceDefinition:
		return "CustomResourceDefinition"
	case MeshConfig:
		return "MeshConfig"
	case MeshNetworks:
		return "MeshNetworks"
	case Namespace:
		return "Namespace"
	case Pod:
		return "Pod"
	case Secret:
		return "Secret"
	case Service:
		return "Service"
	case ServiceAccount:
		return "ServiceAccount"
	case ServiceEntry:
		return "ServiceEntry"
	case StatefulSet:
		return "StatefulSet"
	case ValidatingWebhookConfiguration:
		return "ValidatingWebhookConfiguration"
	case MutatingWebhookConfiguration:
		return "MutatingWebhookConfiguration"
	case PeerAuthentication:
		return "PeerAuthentication"
	case ServiceRoute:
		return "serviceRoute"
	case SubsetRule:
		return "SubsetRule"
	default:
		return "Unknown"
	}
}
