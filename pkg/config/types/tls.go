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

package types

import (
	"crypto/tls"
	"sort"
)

var versions = map[string]uint16{
	"TLSv1_2": tls.VersionTLS12,
	"TLSv1_3": tls.VersionTLS13,
}

var (
	versionNames         []string
	secureCiphers        map[string]uint16
	secureCiphersNames   []string
	insecureCiphers      map[string]uint16
	insecureCiphersNames []string
)

func init() {
	secureCiphers = map[string]uint16{}
	for _, v := range tls.CipherSuites() {
		secureCiphers[v.Name] = v.ID
		secureCiphersNames = append(secureCiphersNames, v.Name)
	}
	insecureCiphers = map[string]uint16{}
	for _, v := range tls.InsecureCipherSuites() {
		insecureCiphers[v.Name] = v.ID
		insecureCiphersNames = append(insecureCiphersNames, v.Name)
	}
	for k := range versions {
		versionNames = append(versionNames, k)
	}
	sort.Strings(versionNames)
}
