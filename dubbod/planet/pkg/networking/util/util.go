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

package util

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"strings"

	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/model"
	dubbonetworking "github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/networking"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
)

const (
	// PassthroughFilterChain to catch traffic that doesn't match other filter chains.
	PassthroughFilterChain = "PassthroughFilterChain"
)

func BuildAdditionalAddresses(extrAddresses []string, listenPort uint32) []*listener.AdditionalAddress {
	var additionalAddresses []*listener.AdditionalAddress
	if len(extrAddresses) > 0 {
		for _, exbd := range extrAddresses {
			if exbd == "" {
				continue
			}
			extraAddress := &listener.AdditionalAddress{
				Address: BuildAddress(exbd, listenPort),
			}
			additionalAddresses = append(additionalAddresses, extraAddress)
		}
	}
	return additionalAddresses
}

func BuildAddress(bind string, port uint32) *core.Address {
	address := BuildNetworkAddress(bind, port, dubbonetworking.TransportProtocolTCP)
	if address != nil {
		return address
	}

	return &core.Address{
		Address: &core.Address_Pipe{
			Pipe: &core.Pipe{
				Path: strings.TrimPrefix(bind, model.UnixAddressPrefix),
			},
		},
	}
}

func BuildNetworkAddress(bind string, port uint32, transport dubbonetworking.TransportProtocol) *core.Address {
	if port == 0 {
		return nil
	}
	return &core.Address{
		Address: &core.Address_SocketAddress{
			SocketAddress: &core.SocketAddress{
				Address:  bind,
				Protocol: transport.ToEnvoySocketProtocol(),
				PortSpecifier: &core.SocketAddress_PortValue{
					PortValue: port,
				},
			},
		},
	}
}

func DelimitedStatsPrefix(statPrefix string) string {
	statPrefix += constants.StatPrefixDelimiter
	return statPrefix
}

// ByteCount returns a human readable byte format
// Inspired by https://yourbasic.org/golang/formatting-byte-size-to-human-readable-format/
func ByteCount(b int) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}
