//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package protocol

import "strings"

// Instance defines network protocols for ports
type Instance string

func (i Instance) String() string {
	return string(i)
}

const (
	// GRPC declares that the port carries gRPC traffic.
	GRPC Instance = "GRPC"
	// GRPCWeb declares that the port carries gRPC traffic.
	GRPCWeb Instance = "GRPC-Web"
	// HTTP2 declares that the port carries HTTP/2 traffic.
	HTTP2 Instance = "HTTP2"
	// HTTPS declares that the port carries HTTPS traffic.
	HTTPS Instance = "HTTPS"
	// TLS declares that the port carries TLS traffic.
	// TLS traffic is assumed to contain SNI as part of the handshake.
	TLS Instance = "TLS"
	// Unsupported - value to signify that the protocol is unsupported.
	Unsupported Instance = "UnsupportedProtocol"
)

// Parse from string ignoring case
func Parse(s string) Instance {
	switch strings.ToLower(s) {
	case "grpc":
		return GRPC
	case "grpc-web":
		return GRPCWeb
	case "http2":
		return HTTP2
	case "https":
		return HTTPS
	case "tls":
		return TLS
	}

	return Unsupported
}
