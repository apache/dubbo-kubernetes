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

package xds

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collections"
)

// SyncStatus reports the xDS synchronization state of a single connected proxy.
// A resource type is SYNCED when the last nonce sent by dubbod has been ACKed.
type SyncStatus struct {
	ProxyID      string              `json:"proxy"`
	ClusterID    string              `json:"cluster_id,omitempty"`
	DubboVersion string              `json:"dubbo_version,omitempty"`
	ConnectedAt  time.Time           `json:"connected_at"`
	WatchedTypes []WatchedTypeStatus `json:"watched_types"`
}

// WatchedTypeStatus reports nonce state for one watched xDS type.
type WatchedTypeStatus struct {
	TypeURL      string    `json:"type_url"`
	NonceSent    string    `json:"nonce_sent,omitempty"`
	NonceAcked   string    `json:"nonce_acked,omitempty"`
	Synced       bool      `json:"synced"`
	ResourceSize int       `json:"resources"`
	LastSendTime time.Time `json:"last_send_time,omitzero"`
	LastError    string    `json:"last_error,omitempty"`
}

type debugHandler struct {
	path string
	help string
}

// AppendDebugHandlers registers the dubbod debug endpoints on the given mux.
// These power `dubboctl proxy-status` and manual troubleshooting.
func (s *DiscoveryServer) AppendDebugHandlers(mux *http.ServeMux) {
	handlers := []debugHandler{}
	register := func(path, help string, h http.HandlerFunc) {
		mux.HandleFunc(path, h)
		handlers = append(handlers, debugHandler{path: path, help: help})
	}

	register("/debug/syncz", "Synchronization status of all proxies connected to this dubbod instance", s.syncz)
	register("/debug/configz", "Configuration resources known to this dubbod instance", s.configz)
	register("/debug/registryz", "Services in the service registry", s.registryz)

	mux.HandleFunc("/debug", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, handlers)
	})
}

// Syncz returns the sync status of all connected proxies. Exported for reuse in tests.
func (s *DiscoveryServer) Syncz() []SyncStatus {
	statuses := make([]SyncStatus, 0, s.adsClientCount())
	for _, con := range s.AllClients() {
		proxy := con.Proxy()
		if proxy == nil {
			continue
		}
		status := SyncStatus{
			ProxyID:     proxy.ID,
			ConnectedAt: con.ConnectedAt(),
		}
		if proxy.Metadata != nil {
			status.ClusterID = string(proxy.Metadata.ClusterID)
		}
		proxy.RLock()
		for typeURL, wr := range proxy.WatchedResources {
			if wr == nil {
				continue
			}
			status.WatchedTypes = append(status.WatchedTypes, WatchedTypeStatus{
				TypeURL:      typeURL,
				NonceSent:    wr.NonceSent,
				NonceAcked:   wr.NonceAcked,
				Synced:       wr.NonceSent != "" && wr.NonceSent == wr.NonceAcked,
				ResourceSize: len(wr.ResourceNames),
				LastSendTime: wr.LastSendTime,
				LastError:    wr.LastError,
			})
		}
		proxy.RUnlock()
		sort.Slice(status.WatchedTypes, func(i, j int) bool {
			return status.WatchedTypes[i].TypeURL < status.WatchedTypes[j].TypeURL
		})
		statuses = append(statuses, status)
	}
	sort.Slice(statuses, func(i, j int) bool { return statuses[i].ProxyID < statuses[j].ProxyID })
	return statuses
}

func (s *DiscoveryServer) syncz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, s.Syncz())
}

func (s *DiscoveryServer) configz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.Env == nil || s.Env.ConfigStore == nil {
		writeJSON(w, []any{})
		return
	}
	configs := make([]map[string]any, 0)
	for _, schema := range collections.Dubbo.All() {
		cfgs := s.Env.List(schema.GroupVersionKind(), "")
		for _, cfg := range cfgs {
			configs = append(configs, map[string]any{
				"kind":      cfg.GroupVersionKind.Kind,
				"group":     cfg.GroupVersionKind.Group,
				"name":      cfg.Name,
				"namespace": cfg.Namespace,
			})
		}
	}
	writeJSON(w, configs)
}

func (s *DiscoveryServer) registryz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	push := s.Env.PushContext()
	if push == nil {
		writeJSON(w, []any{})
		return
	}
	services := make([]map[string]any, 0)
	for _, svc := range push.GetAllServices() {
		if svc == nil {
			continue
		}
		services = append(services, map[string]any{
			"hostname":  string(svc.Hostname),
			"namespace": svc.Attributes.Namespace,
			"ports":     svc.Ports,
		})
	}
	writeJSON(w, services)
}

func writeJSON(w http.ResponseWriter, obj any) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(obj); err != nil {
		http.Error(w, fmt.Sprintf("marshal failure: %v", err), http.StatusInternalServerError)
	}
}
