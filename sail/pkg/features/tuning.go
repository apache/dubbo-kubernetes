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
	"runtime"
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

	RequestLimit = func() float64 {
		v := env.Register(
			"SAIL_MAX_REQUESTS_PER_SECOND",
			0.0,
			"Limits the number of incoming XDS requests per second. On larger machines this can be increased to handle more proxies concurrently. "+
				"If set to 0 or unset, the max will be automatically determined based on the machine size",
		).Get()
		if v > 0 {
			return v
		}
		procs := runtime.GOMAXPROCS(0)
		// Heuristic to scale with cores. We end up with...
		// 1: 20
		// 2: 25
		// 4: 35
		// 32: 100
		return min(float64(15+5*procs), 100.0)
	}()

	DebounceAfter = env.Register(
		"SAIL_DEBOUNCE_AFTER",
		100*time.Millisecond,
		"The delay added to config/registry events for debouncing. This will delay the push by "+
			"at least this interval. If no change is detected within this period, the push will happen, "+
			" otherwise we'll keep delaying until things settle, up to a max of PILOT_DEBOUNCE_MAX.",
	).Get()

	DebounceMax = env.Register(
		"SAIL_DEBOUNCE_MAX",
		10*time.Second,
		"The maximum amount of time to wait for events while debouncing. If events keep showing up with no breaks "+
			"for this time, we'll trigger a push.",
	).Get()

	EnableEDSDebounce = env.Register(
		"SAIL_ENABLE_EDS_DEBOUNCE",
		true,
		"If enabled, Sail will include EDS pushes in the push debouncing, configured by PILOT_DEBOUNCE_AFTER and PILOT_DEBOUNCE_MAX."+
			" EDS pushes may be delayed, but there will be fewer pushes. By default this is enabled",
	).Get()

	PushThrottle = func() int {
		v := env.Register(
			"PILOT_PUSH_THROTTLE",
			0,
			"Limits the number of concurrent pushes allowed. On larger machines this can be increased for faster pushes. "+
				"If set to 0 or unset, the max will be automatically determined based on the machine size",
		).Get()
		if v > 0 {
			return v
		}
		procs := runtime.GOMAXPROCS(0)
		// Heuristic to scale with cores. We end up with...
		// 1: 20
		// 2: 25
		// 4: 35
		// 32: 100
		return min(15+5*procs, 100)
	}()
)
