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
	MySQL      MySQL  `yaml:"mysql"`
}

func (bufman *Bufman) Sanitize() {
}

func (bufman *Bufman) Validate() error {
	// TODO Validate bufman
	return nil
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

type MySQL struct {
	MysqlDsn           string        `yaml:"mysql_dsn"`
	MaxOpenConnections int           `yaml:"max_open_connections"`
	MaxIdleConnections int           `yaml:"max_idle_connections"`
	MaxLifeTime        time.Duration `yaml:"max_life_time"`
	MaxIdleTime        time.Duration `yaml:"max_idle_time"`
}

func (mysql *MySQL) Sanitize() {
}

func (mysql *MySQL) Validate() error {
	// TODO Validate mysql
	return nil
}
