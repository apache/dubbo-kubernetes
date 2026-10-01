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

package model

import (
	"testing"

	"github.com/apache/dubbo-kubernetes/pkg/slices"
)

func TestServiceUsesLocalDualStackAddresses(t *testing.T) {
	service := &Service{
		Addresses:      []string{"10.0.0.1", "fd00::1"},
		DefaultAddress: "10.0.0.1",
	}
	proxy := &Proxy{}
	if got := service.GetAddressForProxy(proxy); got != "10.0.0.1" {
		t.Fatalf("primary address = %q", got)
	}
	extra := service.GetExtraAddressesForProxy(proxy)
	if !slices.Equal(extra, []string{"fd00::1"}) {
		t.Fatalf("extra addresses = %v", extra)
	}
	extra[0] = "fd00::2"
	if service.Addresses[1] != "fd00::1" {
		t.Fatal("returned addresses share service storage")
	}
	copy := service.DeepCopy()
	if !service.Equals(copy) {
		t.Fatal("service copy differs from original")
	}
	copy.Addresses[1] = "fd00::2"
	if service.Equals(copy) || service.Addresses[1] != "fd00::1" {
		t.Fatal("address changes must affect equality without mutating original")
	}
}

func TestServiceUsesDefaultAddressWithoutVIPs(t *testing.T) {
	service := &Service{DefaultAddress: "10.0.0.1"}
	if got := service.GetAddressForProxy(&Proxy{}); got != service.DefaultAddress {
		t.Fatalf("address = %q, want default address", got)
	}
}
