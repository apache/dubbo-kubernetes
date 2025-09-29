/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ca

import (
	"github.com/apache/dubbo-kubernetes/pkg/security"
	"github.com/apache/dubbo-kubernetes/security/pkg/pki/ca"
	"github.com/apache/dubbo-kubernetes/security/pkg/pki/util"
	"google.golang.org/grpc"
	pb "istio.io/api/security/v1alpha1"
	"time"
)

type Server struct {
	pb.UnimplementedIstioCertificateServiceServer
	Authenticators []security.Authenticator
	serverCertTTL  time.Duration
	ca             CertificateAuthority
}

func New(ca CertificateAuthority, ttl time.Duration, authenticators []security.Authenticator) (*Server, error) {
	certBundle := ca.GetCAKeyCertBundle()
	if len(certBundle.GetRootCertPem()) != 0 {
		RecordCertsExpiry(certBundle)
	}

	server := &Server{
		Authenticators: authenticators,
		serverCertTTL:  ttl,
		ca:             ca,
	}

	return server, nil
}

type CertificateAuthority interface {
	Sign(csrPEM []byte, opts ca.CertOpts) ([]byte, error)
	SignWithCertChain(csrPEM []byte, opts ca.CertOpts) ([]string, error)
	GetCAKeyCertBundle() *util.KeyCertBundle
}

func RecordCertsExpiry(keyCertBundle *util.KeyCertBundle) {}

func (s *Server) Register(grpcServer *grpc.Server) {
	pb.RegisterIstioCertificateServiceServer(grpcServer, s)
}
