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

package protocol

import (
	coremesh "github.com/apache/dubbo-kubernetes/pkg/core/resource/apis/mesh"
)

// protocolStack is a mapping between a protocol and its full protocol stack, e.g.
// HTTP has a protocol stack [HTTP, TCP],
// GRPC has a protocol stack [GRPC, HTTP2, TCP],
// TCP  has a protocol stack [TCP].
var protocolStacks = map[coremesh.Protocol]coremesh.ProtocolList{
	coremesh.ProtocolTriple: {coremesh.ProtocolTriple, coremesh.ProtocolGRPC, coremesh.ProtocolHTTP2, coremesh.ProtocolHTTP, coremesh.ProtocolTCP},
	coremesh.ProtocolGRPC:   {coremesh.ProtocolGRPC, coremesh.ProtocolHTTP2, coremesh.ProtocolTCP},
	coremesh.ProtocolHTTP2:  {coremesh.ProtocolHTTP2, coremesh.ProtocolTCP},
	coremesh.ProtocolHTTP:   {coremesh.ProtocolHTTP, coremesh.ProtocolTCP},
	coremesh.ProtocolKafka:  {coremesh.ProtocolKafka, coremesh.ProtocolTCP},
	coremesh.ProtocolTCP:    {coremesh.ProtocolTCP},
}

// GetCommonProtocol returns a common protocol between given two.
//
// E.g.,
// a common protocol between HTTP and HTTP2 is HTTP2,
// a common protocol between HTTP and HTTP  is HTTP,
// a common protocol between HTTP and TCP   is TCP,
// a common protocol between GRPC and HTTP2 is HTTP2,
// a common protocol between HTTP and HTTP2 is HTTP.
func GetCommonProtocol(one, another coremesh.Protocol) coremesh.Protocol {
	switch {
	case one == another:
		return one
	case one == "" || another == "":
		return coremesh.ProtocolUnknown
	case one == coremesh.ProtocolUnknown || another == coremesh.ProtocolUnknown:
		return coremesh.ProtocolUnknown
	default:
		oneProtocolStack, exist := protocolStacks[one]
		if !exist {
			return coremesh.ProtocolUnknown
		}
		anotherProtocolStack, exist := protocolStacks[another]
		if !exist {
			return coremesh.ProtocolUnknown
		}
		for _, firstProtocol := range oneProtocolStack {
			for _, secondProtocol := range anotherProtocolStack {
				if firstProtocol == secondProtocol {
					return firstProtocol
				}
			}
		}
		return coremesh.ProtocolUnknown
	}
}
