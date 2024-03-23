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

package bufman

import (
	"time"
)

type Bufman struct {
	OpenBufman bool   `yaml:"open_bufman"`
	Server     Server `yaml:"server"`
}

type Server struct {
	ServerHost     string `yaml:"server_host"`
	HTTPPort       int    `yaml:"port"`
	GrpcPlainPort  int    `yaml:"grpc_plain_port"`
	GrpcSecurePort int    `yaml:"grpc_secure_port"`

	PageTokenExpireTime time.Duration `yaml:"page_token_expire_time"`
	PageTokenSecret     string        `yaml:"page_token_secret"`
}

func (s *Server) Sanitize() {
}

func (s *Server) Validate() error {
	// TODO Validate server
	return nil
}

func DefaultBufmanConfig() Bufman {
	return Bufman{
		OpenBufman: false,
		Server: Server{
			ServerHost:          "bufman",
			HTTPPort:            39080,
			GrpcPlainPort:       39091,
			GrpcSecurePort:      39092,
			PageTokenExpireTime: time.Hour,
			PageTokenSecret:     "12345678",
		},
	}
}
