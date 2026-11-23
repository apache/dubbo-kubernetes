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

package nodeagent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	cniconstants "github.com/apache/dubbo-kubernetes/cni/pkg/constants"
	"github.com/apache/dubbo-kubernetes/cni/pkg/plugin"
)

// EventHandler processes CNI ADD and DEL notifications received from dubbo-cni.
type EventHandler interface {
	HandleCNIAdd(ctx context.Context, event plugin.PodEvent) error
	HandleCNIDel(ctx context.Context, event plugin.PodEvent) error
}

// UDSServer accepts HTTP-over-UDS notifications from the chained dubbo-cni binary.
type UDSServer struct {
	socketPath string
	handler    EventHandler
	ln         net.Listener
	srv        *http.Server
}

// NewUDSServer creates a UDSServer bound to socketPath.
func NewUDSServer(socketPath string, handler EventHandler) *UDSServer {
	if socketPath == "" {
		socketPath = cniconstants.DefaultSocketPath
	}
	return &UDSServer{
		socketPath: socketPath,
		handler:    handler,
	}
}

// Start binds the Unix domain socket and serves CNI plugin callbacks until ctx is cancelled.
func (s *UDSServer) Start(ctx context.Context) error {
	if err := os.MkdirAll(filepath.Dir(s.socketPath), 0o755); err != nil {
		return fmt.Errorf("create UDS dir: %w", err)
	}
	_ = os.Remove(s.socketPath)

	ln, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listen UDS %s: %w", s.socketPath, err)
	}
	if err := os.Chmod(s.socketPath, 0o600); err != nil {
		_ = ln.Close()
		return fmt.Errorf("chmod UDS %s: %w", s.socketPath, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/pods/add", s.handleAdd)
	mux.HandleFunc("/v1/pods/del", s.handleDel)

	s.ln = ln
	s.srv = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := s.srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return
		}
	}()
	go func() {
		<-ctx.Done()
		_ = s.Close()
	}()
	return nil
}

// Close stops the Unix socket server and removes the socket file.
func (s *UDSServer) Close() error {
	var err error
	if s.srv != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err = s.srv.Shutdown(shutdownCtx)
		cancel()
	}
	_ = os.Remove(s.socketPath)
	return err
}

func (s *UDSServer) handleAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var ev plugin.PodEvent
	if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
		http.Error(w, fmt.Sprintf("decode event: %v", err), http.StatusBadRequest)
		return
	}
	if err := s.handler.HandleCNIAdd(r.Context(), ev); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *UDSServer) handleDel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var ev plugin.PodEvent
	if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
		http.Error(w, fmt.Sprintf("decode event: %v", err), http.StatusBadRequest)
		return
	}
	if err := s.handler.HandleCNIDel(r.Context(), ev); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
