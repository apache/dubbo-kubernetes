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
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstIntFromEnv(fallback int, names ...string) int {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			if parsed, err := strconv.Atoi(value); err == nil {
				return parsed
			}
		}
	}
	return fallback
}

func autoDiscoverServiceTarget(env []string, namespace, domainSuffix string) (string, int, error) {
	type serviceTarget struct {
		name string
		port int
	}

	ports := map[string]int{}
	hosts := map[string]struct{}{}
	for _, raw := range env {
		key, value, found := strings.Cut(raw, "=")
		if !found {
			continue
		}
		switch {
		case strings.HasSuffix(key, "_SERVICE_HOST"):
			base := strings.TrimSuffix(key, "_SERVICE_HOST")
			hosts[base] = struct{}{}
		case strings.HasSuffix(key, "_SERVICE_PORT"):
			base := strings.TrimSuffix(key, "_SERVICE_PORT")
			port, err := strconv.Atoi(value)
			if err != nil {
				continue
			}
			ports[base] = port
		}
	}

	candidates := make([]serviceTarget, 0)
	for base := range hosts {
		if base == "KUBERNETES" {
			continue
		}
		port, found := ports[base]
		if !found {
			continue
		}
		candidates = append(candidates, serviceTarget{
			name: strings.ToLower(strings.ReplaceAll(base, "_", "-")),
			port: port,
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].name < candidates[j].name
	})
	if len(candidates) == 0 {
		return "", 80, nil
	}
	if len(candidates) > 1 {
		names := make([]string, 0, len(candidates))
		for _, candidate := range candidates {
			names = append(names, candidate.name)
		}
		return "", 80, fmt.Errorf("multiple Services detected in namespace: %s; set --host or DUBBO_SERVICE_HOST", strings.Join(names, ", "))
	}
	host := fmt.Sprintf("%s.%s.svc.%s", candidates[0].name, namespace, domainSuffix)
	return host, candidates[0].port, nil
}
