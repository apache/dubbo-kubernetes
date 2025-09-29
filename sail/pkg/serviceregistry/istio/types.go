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

// Package istio provides Istio service mesh integration for Dubbo service registry.
// It implements the service registry interface to discover and manage services
// through Istio's Pilot discovery service using xDS protocol.
package istio

import (
	"time"
)

// IstioConfig holds configuration for Istio service registry
type IstioConfig struct {
	PilotAddress    string        `json:"pilotAddress"`
	Namespace       string        `json:"namespace"`
	EnableDiscovery bool          `json:"enableDiscovery"`
	TLSEnabled      bool          `json:"tlsEnabled"`
	CertPath        string        `json:"certPath"`
	KeyPath         string        `json:"keyPath"`
	CAPath          string        `json:"caPath"`
	SyncTimeout     time.Duration `json:"syncTimeout"`
}

// IstioServiceInfo represents service information from Istio
type IstioServiceInfo struct {
	Hostname        string            `json:"hostname"`
	Namespace       string            `json:"namespace"`
	Ports           []IstioPortInfo   `json:"ports"`
	Labels          map[string]string `json:"labels"`
	Endpoints       []IstioEndpoint   `json:"endpoints"`
	VirtualService  *VirtualService   `json:"virtualService,omitempty"`
	DestinationRule *DestinationRule  `json:"destinationRule,omitempty"`
}

// IstioPortInfo represents port information for Istio services
type IstioPortInfo struct {
	Name     string `json:"name"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

// IstioEndpoint represents an endpoint for Istio services
type IstioEndpoint struct {
	Address string            `json:"address"`
	Port    int               `json:"port"`
	Labels  map[string]string `json:"labels"`
}

// VirtualService represents Istio VirtualService configuration
type VirtualService struct {
	Name      string                 `json:"name"`
	Namespace string                 `json:"namespace"`
	Spec      map[string]interface{} `json:"spec"`
}

// DestinationRule represents Istio DestinationRule configuration
type DestinationRule struct {
	Name      string                 `json:"name"`
	Namespace string                 `json:"namespace"`
	Spec      map[string]interface{} `json:"spec"`
}

// PilotClient defines the interface for communicating with Istio Pilot
type PilotClient interface {
	Connect() error
	Disconnect() error
	WatchServices(namespace string, handler ServiceEventHandler) error
	GetService(hostname string) (*IstioServiceInfo, error)
	IsConnected() bool
}

// ServiceEventHandler defines the interface for handling service events
type ServiceEventHandler interface {
	OnServiceAdd(service *IstioServiceInfo)
	OnServiceUpdate(oldService, newService *IstioServiceInfo)
	OnServiceDelete(service *IstioServiceInfo)
}

// DefaultConfig returns default Istio configuration
func DefaultConfig() *IstioConfig {
	return &IstioConfig{
		PilotAddress:    "istiod.istio-system.svc.cluster.local:15010",
		Namespace:       "istio-system",
		EnableDiscovery: true,
		TLSEnabled:      true,
		SyncTimeout:     30 * time.Second,
	}
}
