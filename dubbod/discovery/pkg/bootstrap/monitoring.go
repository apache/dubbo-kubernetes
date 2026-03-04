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

package bootstrap

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/log"
	"github.com/apache/dubbo-kubernetes/pkg/monitoring"
	"github.com/apache/dubbo-kubernetes/pkg/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	metricsPath = "/metrics"
	versionPath = "/version"
)

var (
	serverStart = time.Now()

	_ = monitoring.NewDerivedGauge(
		"dubbod_uptime_seconds",
		"Current dubbod server uptime in seconds",
	).ValueFrom(func() float64 {
		return time.Since(serverStart).Seconds()
	})

	versionTag   = monitoring.CreateLabel("version")
	dubbodVersion = monitoring.NewGauge(
		"dubbod_info",
		"Dubbod version and build information.",
		monitoring.WithLabels("version"),
	)
)

func addMonitor(mux *http.ServeMux) {
	exporter := promhttp.HandlerFor(monitoring.GetRegistry(), promhttp.HandlerOpts{})
	mux.Handle(metricsPath, exporter)

	mux.HandleFunc(versionPath, func(out http.ResponseWriter, req *http.Request) {
		if _, err := out.Write([]byte(version.Info.String())); err != nil {
			log.Errorf("Unable to write version string: %v", err)
		}
	})
}

func (s *Server) initMonitor(addr string) error {
	s.addStartFunc("monitoring", func(stop <-chan struct{}) error {
		if addr == "" {
			return nil
		}

		listener, err := net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("unable to listen on socket: %v", err)
		}

		addMonitor(s.monitoringMux)

		// Record version info
		dubbodVersion.With(versionTag.Value(version.Info.String())).Record(1)

		monitoringServer := &http.Server{
			Addr:        listener.Addr().String(),
			Handler:     s.monitoringMux,
			IdleTimeout: 90 * time.Second,
			ReadTimeout: 30 * time.Second,
		}

		go func() {
			log.Infof("starting monitoring server at %s", listener.Addr())
			if err := monitoringServer.Serve(listener); err != nil && err != http.ErrServerClosed {
				log.Errorf("error serving monitoring server: %v", err)
			}
		}()

		go func() {
			<-stop
			if err := monitoringServer.Close(); err != nil {
				log.Errorf("error closing monitoring server: %v", err)
			}
		}()

		return nil
	})
	return nil
}
