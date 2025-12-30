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
	"crypto/tls"
	"log"
	"net/http"
	"strings"

	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"
	sec_model "github.com/apache/dubbo-kubernetes/pkg/model"
)

type httpServerErrorLogWriter struct{}

func (*httpServerErrorLogWriter) Write(p []byte) (int, error) {
	m := strings.TrimSuffix(string(p), "\n")
	if strings.HasPrefix(m, "http: TLS handshake error") && strings.HasSuffix(m, ": EOF") {
		dubbolog.Debug(m)
	} else {
		dubbolog.Info(m)
	}
	return len(p), nil
}

func (s *Server) initSecureWebhookServer(args *PlanetArgs) {
	if args.ServerOptions.HTTPSAddr == "" {
		s.httpsMux = s.httpMux
		dubbolog.Infof("HTTPS port is disabled, multiplexing webhooks on the httpAddr %v", args.ServerOptions.HTTPAddr)
		return
	}

	tlsConfig := &tls.Config{
		GetCertificate: s.getDubbodCertificate,
		MinVersion:     tls.VersionTLS12,
		CipherSuites:   args.ServerOptions.TLSOptions.CipherSuits,
	}
	// Compliance for control plane validation and injection webhook server.
	sec_model.EnforceGoCompliance(tlsConfig)

	dubbolog.Info("initializing secure webhook server for dubbod webhooks")
	// create the https server for hosting the k8s injectionWebhook handlers.
	s.httpsMux = http.NewServeMux()
	s.httpsServer = &http.Server{
		Addr:      args.ServerOptions.HTTPSAddr,
		ErrorLog:  log.New(&httpServerErrorLogWriter{}, "", 0),
		Handler:   s.httpsMux,
		TLSConfig: tlsConfig,
	}

	s.httpsMux.HandleFunc("/ready", s.dubbodReadyHandler)
}
