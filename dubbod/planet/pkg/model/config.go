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
	"cmp"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	netutil "github.com/apache/dubbo-kubernetes/pkg/util/net"
	"sort"
	"strings"

	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collection"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/util/hash"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
)

const (
	NamespaceAll = ""
)

type (
	ConfigHash   uint64
	EventHandler = func(config.Config, config.Config, Event)
)

type ConfigStore interface {
	Schemas() collection.Schemas
	Get(typ config.GroupVersionKind, name, namespace string) *config.Config
	List(typ config.GroupVersionKind, namespace string) []config.Config
	Create(config config.Config) (revision string, err error)
	Update(config config.Config) (newRevision string, err error)
	UpdateStatus(config config.Config) (newRevision string, err error)
	Patch(orig config.Config, patchFn config.PatchFunc) (string, error)
	Delete(typ config.GroupVersionKind, name, namespace string, resourceVersion *string) error
}

type ConfigStoreController interface {
	ConfigStore
	RegisterEventHandler(kind config.GroupVersionKind, handler EventHandler)
	Run(stop <-chan struct{})
	HasSynced() bool
}

func sortConfigByCreationTime(configs []config.Config) []config.Config {
	sort.Slice(configs, func(i, j int) bool {
		if r := configs[i].CreationTimestamp.Compare(configs[j].CreationTimestamp); r != 0 {
			return r == -1 // -1 means i is less than j, so return true
		}
		if r := cmp.Compare(configs[i].Name, configs[j].Name); r != 0 {
			return r == -1
		}
		return cmp.Compare(configs[i].Namespace, configs[j].Namespace) == -1
	})
	return configs
}

func (key ConfigKey) String() string {
	return key.Kind.String() + "/" + key.Namespace + "/" + key.Name
}

func (key ConfigKey) HashCode() ConfigHash {
	h := hash.New()
	h.Write([]byte{byte(key.Kind)})
	// Add separator / to avoid collision.
	h.WriteString("/")
	h.WriteString(key.Namespace)
	h.WriteString("/")
	h.WriteString(key.Name)
	return ConfigHash(h.Sum64())
}

func ResolveShortnameToFQDN(hostname string, meta config.Meta) host.Name {
	if len(hostname) == 0 {
		// only happens when the gateway-api BackendRef is invalid
		return ""
	}
	out := hostname
	// Treat the wildcard hostname as fully qualified. Any other variant of a wildcard hostname will contain a `.` too,
	// and skip the next if, so we only need to check for the literal wildcard itself.
	if hostname == "*" {
		return host.Name(out)
	}

	// if the hostname is a valid ipv4 or ipv6 address, do not append domain or namespace
	if netutil.IsValidIPAddress(hostname) {
		return host.Name(out)
	}

	// if FQDN is specified, do not append domain or namespace to hostname
	if !strings.Contains(hostname, ".") {
		if meta.Namespace != "" {
			out = out + "." + meta.Namespace
		}

		// FIXME this is a gross hack to hardcode a service's domain name in kubernetes
		// BUG this will break non kubernetes environments if they use shortnames in the
		// rules.
		if meta.Domain != "" {
			out = out + ".svc." + meta.Domain
		}
	}

	return host.Name(out)
}

func HasConfigsOfKind(configs sets.Set[ConfigKey], kind kind.Kind) bool {
	for c := range configs {
		if c.Kind == kind {
			return true
		}
	}
	return false
}
