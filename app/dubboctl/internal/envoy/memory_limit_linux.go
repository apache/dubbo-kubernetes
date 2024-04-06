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
	"os"
	"strconv"
	"strings"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/proxy/cgroups"
)

type UIntOrString struct {
	Type   string
	UInt   uint64
	String string
}

func DetectMaxMemory() uint64 {
	switch cgroups.Mode() {
	case cgroups.Legacy:
		res := maybeReadAsBytes("/sys/fs/cgroup/memory.limit_in_bytes")
		if res != nil && res.Type == "int" {
			return res.UInt
		}
	case cgroups.Hybrid, cgroups.Unified:
		res := maybeReadAsBytes("/sys/fs/cgroup/memory.max")
		if res != nil && res.Type == "int" {
			return res.UInt
		}
	}
	return 0
}

func maybeReadAsBytes(path string) *UIntOrString {
	byteContents, err := os.ReadFile(path)
	if err == nil {
		contents := strings.TrimSpace(string(byteContents))
		bytes, err := strconv.ParseUint(contents, 10, 64)
		if err != nil {
			return &UIntOrString{
				Type:   "string",
				String: contents,
			}
		}
		return &UIntOrString{
			Type: "int",
			UInt: bytes,
		}
	}
	return nil
}
