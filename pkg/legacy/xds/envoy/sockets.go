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

package envoy

import (
	"fmt"
)

// AccessLogSocketName generates a socket path that will fit the Unix socket path limitation of 108 chars
func AccessLogSocketName(name, mesh string) string {
	return socketName(fmt.Sprintf("/tmp/dubbo-al-%s-%s", name, mesh))
}

// MetricsHijackerSocketName generates a socket path that will fit the Unix socket path limitation of 108 chars
func MetricsHijackerSocketName(name, mesh string) string {
	return socketName(fmt.Sprintf("/tmp/dubbo-mh-%s-%s", name, mesh))
}

func socketName(s string) string {
	trimLen := len(s)
	if trimLen > 100 {
		trimLen = 100
	}
	return s[:trimLen] + ".sock"
}
