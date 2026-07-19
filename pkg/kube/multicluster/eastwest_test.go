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

package multicluster

import (
	"testing"

	"github.com/apache/dubbo-kubernetes/pkg/cluster"
)

func TestParseEastWestGateways(t *testing.T) {
	got, err := ParseEastWestGateways("remote=192.168.15.155:15443,primary=dubbod-eastwest.example.com:443")
	if err != nil {
		t.Fatalf("ParseEastWestGateways() error = %v", err)
	}
	if got[cluster.ID("remote")].Address != "192.168.15.155" || got[cluster.ID("remote")].Port != 15443 {
		t.Fatalf("remote gateway = %#v", got[cluster.ID("remote")])
	}
	if got[cluster.ID("primary")].Endpoint() != "dubbod-eastwest.example.com:443" {
		t.Fatalf("primary endpoint = %q", got[cluster.ID("primary")].Endpoint())
	}
}

func TestParseEastWestGatewaysRejectsBadEntries(t *testing.T) {
	for _, raw := range []string{
		"remote",
		"remote=192.168.15.155",
		"remote=192.168.15.155:notaport",
		"remote=192.168.15.155:0",
	} {
		t.Run(raw, func(t *testing.T) {
			if _, err := ParseEastWestGateways(raw); err == nil {
				t.Fatal("ParseEastWestGateways() returned nil error")
			}
		})
	}
}
