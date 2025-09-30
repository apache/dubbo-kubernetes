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

package features

import (
	"github.com/apache/dubbo-kubernetes/pkg/env"
	"time"
)

var (
	MaxConcurrentStreams = env.Register(
		"DUBBO_GPRC_MAXSTREAMS",
		100000,
		"Sets the maximum number of concurrent grpc streams.",
	).Get()

	// MaxRecvMsgSize The max receive buffer size of gRPC received channel of Pilot in bytes.
	MaxRecvMsgSize = env.Register(
		"DUBBO_GPRC_MAXRECVMSGSIZE",
		4*1024*1024,
		"Sets the max receive buffer size of gRPC stream in bytes.",
	).Get()

	XDSCacheMaxSize = env.Register("SAIL_XDS_CACHE_SIZE", 60000,
		"The maximum number of cache entries for the XDS cache.").Get()
	XDSCacheIndexClearInterval = env.Register("SAIL_XDS_CACHE_INDEX_CLEAR_INTERVAL", 5*time.Second,
		"The interval for xds cache index clearing.").Get()
)
