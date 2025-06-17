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

package mesh

import (
	"hash/fnv"
	"net"
	"strconv"
	"strings"

	coremodel "github.com/apache/dubbo-kubernetes/pkg/core/model"
)

// Protocol identifies a protocol supported by a service.
type Protocol string

const (
	ProtocolUnknown = "<unknown>"
	ProtocolTCP     = "tcp"
	ProtocolHTTP    = "http"
	ProtocolHTTP2   = "http2"
	ProtocolGRPC    = "grpc"
	ProtocolKafka   = "kafka"
	ProtocolTriple  = "triple"
)

func ParseProtocol(tag string) Protocol {
	switch strings.ToLower(tag) {
	case ProtocolHTTP:
		return ProtocolHTTP
	case ProtocolHTTP2:
		return ProtocolHTTP2
	case ProtocolTCP:
		return ProtocolTCP
	case ProtocolGRPC:
		return ProtocolGRPC
	case ProtocolKafka:
		return ProtocolKafka
	case ProtocolTriple:
		return ProtocolTriple
	default:
		return ProtocolUnknown
	}
}

// ProtocolList represents a list of Protocols.
type ProtocolList []Protocol

func (l ProtocolList) Strings() []string {
	values := make([]string, len(l))
	for i := range l {
		values[i] = string(l[i])
	}
	return values
}

// SupportedProtocols is a list of supported protocols that will be communicated to a user.
var SupportedProtocols = ProtocolList{
	ProtocolGRPC,
	ProtocolHTTP,
	ProtocolHTTP2,
	ProtocolKafka,
	ProtocolTCP,
}

// Service that indicates L4 pass through cluster
const PassThroughService = "pass_through"

var (
	IPv4Loopback = net.IPv4(127, 0, 0, 1)
	IPv6Loopback = net.IPv6loopback
)

func (d *DataplaneResource) UsesInterface(address net.IP, port uint32) bool {
	return d.UsesInboundInterface(address, port) || d.UsesOutboundInterface(address, port)
}

func (d *DataplaneResource) UsesInboundInterface(address net.IP, port uint32) bool {
	if d == nil {
		return false
	}
	for _, iface := range d.Spec.Networking.GetInboundInterfaces() {
		// compare against port and IP address of the dataplane
		if port == iface.DataplanePort && overlap(address, net.ParseIP(iface.DataplaneIP)) {
			return true
		}
		// compare against port and IP address of the application
		if port == iface.WorkloadPort && overlap(address, net.ParseIP(iface.WorkloadIP)) {
			return true
		}
	}
	return false
}

func (d *DataplaneResource) UsesOutboundInterface(address net.IP, port uint32) bool {
	if d == nil {
		return false
	}
	for _, oface := range d.Spec.Networking.GetOutboundInterfaces() {
		// compare against port and IP address of the dataplane
		if port == oface.DataplanePort && overlap(address, net.ParseIP(oface.DataplaneIP)) {
			return true
		}
	}
	return false
}

func overlap(address1 net.IP, address2 net.IP) bool {
	if address1.IsUnspecified() || address2.IsUnspecified() {
		// wildcard match (either IPv4 address "0.0.0.0" or the IPv6 address "::")
		return true
	}
	// exact match
	return address1.Equal(address2)
}

func (d *DataplaneResource) GetIP() string {
	if d == nil {
		return ""
	}
	if d.Spec.Networking.AdvertisedAddress != "" {
		ip := strings.Split(d.Spec.Networking.Address, ":")
		return ip[0]
		return d.Spec.Networking.AdvertisedAddress
	} else {
		ip := strings.Split(d.Spec.Networking.Address, ":")
		return ip[0]
	}
}

func (d *DataplaneResource) IsIPv6() bool {
	if d == nil {
		return false
	}

	ip := net.ParseIP(d.Spec.Networking.Address)
	if ip == nil {
		return false
	}

	return ip.To4() == nil
}

func (d *DataplaneResource) AdminAddress(defaultAdminPort uint32) string {
	if d == nil {
		return ""
	}
	ip := d.GetIP()
	adminPort := d.AdminPort(defaultAdminPort)
	return net.JoinHostPort(ip, strconv.FormatUint(uint64(adminPort), 10))
}

func (d *DataplaneResource) AdminPort(defaultAdminPort uint32) uint32 {
	if d == nil {
		return 0
	}
	if adminPort := d.Spec.GetNetworking().GetAdmin().GetPort(); adminPort != 0 {
		return adminPort
	}
	return defaultAdminPort
}

func (d *DataplaneResource) Hash() []byte {
	hasher := fnv.New128a()
	_, _ = hasher.Write(coremodel.HashMeta(d))
	_, _ = hasher.Write([]byte(d.Spec.GetNetworking().GetAddress()))
	_, _ = hasher.Write([]byte(d.Spec.GetNetworking().GetAdvertisedAddress()))
	return hasher.Sum(nil)
}
