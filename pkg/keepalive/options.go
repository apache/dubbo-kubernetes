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

package keepalive

import (
	"github.com/apache/dubbo-kubernetes/pkg/env"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"math"
	"time"
)

const (
	Infinity = time.Duration(math.MaxInt64)
)

var (
	grpcKeepaliveInterval = env.Register("GRPC_KEEPALIVE_INTERVAL", 30*time.Second, "gRPC Keepalive Interval").Get()
	grpcKeepaliveTimeout  = env.Register("GRPC_KEEPALIVE_TIMEOUT", 10*time.Second, "gRPC Keepalive Timeout").Get()
)

type Options struct {
	Time                        time.Duration
	Timeout                     time.Duration
	MaxServerConnectionAge      time.Duration // default value is infinity
	MaxServerConnectionAgeGrace time.Duration // default value 10s
}

func (o *Options) ConvertToClientOption() grpc.DialOption {
	return grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:    o.Time,
		Timeout: o.Timeout,
	})
}

func DefaultOption() *Options {
	return &Options{
		Time:                        grpcKeepaliveInterval,
		Timeout:                     grpcKeepaliveTimeout,
		MaxServerConnectionAge:      Infinity,
		MaxServerConnectionAgeGrace: 10 * time.Second,
	}
}
