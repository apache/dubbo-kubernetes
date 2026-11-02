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

package app

import (
	"strings"
	"testing"
)

func TestAutoDiscoverServiceTargetSingleService(t *testing.T) {
	host, port, err := autoDiscoverServiceTarget([]string{
		"KUBERNETES_SERVICE_HOST=10.96.0.1",
		"KUBERNETES_SERVICE_PORT=443",
		"NGINX_SERVICE_HOST=10.96.12.34",
		"NGINX_SERVICE_PORT=80",
	}, "app", "cluster.local")
	if err != nil {
		t.Fatalf("autoDiscoverServiceTarget() error = %v", err)
	}
	if host != "nginx.app.svc.cluster.local" {
		t.Fatalf("host = %q, want %q", host, "nginx.app.svc.cluster.local")
	}
	if port != 80 {
		t.Fatalf("port = %d, want 80", port)
	}
}

func TestAutoDiscoverServiceTargetMultipleServices(t *testing.T) {
	_, _, err := autoDiscoverServiceTarget([]string{
		"NGINX_SERVICE_HOST=10.96.12.34",
		"NGINX_SERVICE_PORT=80",
		"REDIS_SERVICE_HOST=10.96.22.44",
		"REDIS_SERVICE_PORT=6379",
	}, "app", "cluster.local")
	if err == nil {
		t.Fatalf("autoDiscoverServiceTarget() error = nil, want multiple services error")
	}
	if !strings.Contains(err.Error(), "multiple Services detected") {
		t.Fatalf("error = %q, want multiple services message", err.Error())
	}
}
