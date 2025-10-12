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

// Package annotations provides utilities for parsing Dubbo service annotations
// from Kubernetes Service objects. It supports standard Dubbo service discovery
// annotations that enable Kubernetes Services to be discovered as Dubbo services.
//
// The parser supports the following annotations:
// - dubbo.apache.org/service-name: The interface name of the Dubbo service
// - dubbo.apache.org/version: The version of the Dubbo service (optional)
// - dubbo.apache.org/group: The group of the Dubbo service (optional)
// - dubbo.apache.org/protocol: The protocol used by the Dubbo service (optional)
// - dubbo.apache.org/port: The port number for the Dubbo service (optional)
//
// Example usage:
//
//	k8sService := &corev1.Service{...}
//	if IsDubboService(k8sService) {
//	    dubboInfo, err := ParseDubboAnnotations(k8sService)
//	    if err == nil {
//	        // Process dubbo service
//	    }
//	}
package annotations

import (
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
)

// Dubbo service annotation constants
const (
	DubboServiceNameAnnotation = "dubbo.apache.org/service-name"
	DubboVersionAnnotation     = "dubbo.apache.org/version"
	DubboGroupAnnotation       = "dubbo.apache.org/group"
	DubboProtocolAnnotation    = "dubbo.apache.org/protocol"
	DubboPortAnnotation        = "dubbo.apache.org/port"
)

// DubboServiceInfo contains parsed Dubbo service information from k8s annotations
type DubboServiceInfo struct {
	ServiceName string
	Version     string
	Group       string
	Protocol    string
	Port        int32
}

// ParseDubboAnnotations parses Dubbo-related annotations from a Kubernetes Service
func ParseDubboAnnotations(service *corev1.Service) (*DubboServiceInfo, error) {
	if service == nil {
		return nil, fmt.Errorf("service is nil")
	}

	annotations := service.GetAnnotations()
	if annotations == nil {
		return nil, fmt.Errorf("no annotations found")
	}

	// Check if this service has Dubbo service name annotation
	serviceName, hasServiceName := annotations[DubboServiceNameAnnotation]
	if !hasServiceName || serviceName == "" {
		return nil, fmt.Errorf("missing or empty %s annotation", DubboServiceNameAnnotation)
	}

	info := &DubboServiceInfo{
		ServiceName: serviceName,
		Version:     "1.0.0",   // default version
		Group:       "default", // default group
		Protocol:    "dubbo",   // default protocol
	}

	// Parse optional annotations
	if version, ok := annotations[DubboVersionAnnotation]; ok && version != "" {
		info.Version = version
	}

	if group, ok := annotations[DubboGroupAnnotation]; ok && group != "" {
		info.Group = group
	}

	if protocol, ok := annotations[DubboProtocolAnnotation]; ok && protocol != "" {
		info.Protocol = protocol
	}

	// Parse port annotation if specified
	if portStr, ok := annotations[DubboPortAnnotation]; ok && portStr != "" {
		port, err := strconv.ParseInt(portStr, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid port annotation %s: %v", portStr, err)
		}
		info.Port = int32(port)
	}

	return info, nil
}

// IsDubboService checks if a Kubernetes Service has Dubbo annotations
func IsDubboService(service *corev1.Service) bool {
	if service == nil {
		return false
	}

	annotations := service.GetAnnotations()
	if annotations == nil {
		return false
	}

	_, hasDubboAnnotation := annotations[DubboServiceNameAnnotation]
	return hasDubboAnnotation
}
