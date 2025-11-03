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
	"context"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/security"
	"github.com/apache/dubbo-kubernetes/security/pkg/pki/ca"
	caerror "github.com/apache/dubbo-kubernetes/security/pkg/pki/error"
	"github.com/apache/dubbo-kubernetes/security/pkg/pki/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pb "istio.io/api/security/v1alpha1"
	"k8s.io/klog/v2"
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

// CreateCertificate handles an incoming certificate signing request (CSR). It does
// authentication and authorization. Upon validated, signs a certificate that:
// the SAN is the identity of the caller in authentication result.
// the subject public key is the public key in the CSR.
// the validity duration is the ValidityDuration in request, or default value if the given duration is invalid.
// it is signed by the CA signing key.
func (s *Server) CreateCertificate(ctx context.Context, request *pb.IstioCertificateRequest) (
	*pb.IstioCertificateResponse, error,
) {
	caller, err := s.authenticate(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "request authenticate failure")
	}

	// By default, we will use the callers identity for the certificate
	var sans []string
	if caller != nil {
		sans = caller.Identities
	}
	crMetadata := request.Metadata.GetFields()
	certSigner := crMetadata[security.CertSigner].GetStringValue()
	_, _, certChainBytes, rootCertBytes := s.ca.GetCAKeyCertBundle().GetAll()
	certOpts := ca.CertOpts{
		SubjectIDs: sans,
		TTL:        time.Duration(request.ValidityDuration) * time.Second,
		ForCA:      false,
		CertSigner: certSigner,
	}
	var signErr error
	var cert []byte
	var respCertChain []string
	if certSigner == "" {
		cert, signErr = s.ca.Sign([]byte(request.Csr), certOpts)
	} else {
		respCertChain, signErr = s.ca.SignWithCertChain([]byte(request.Csr), certOpts)
	}
	if signErr != nil {
		klog.Errorf("CSR signing error: %v", signErr.Error())
		if caErr, ok := signErr.(*caerror.Error); ok {
			return nil, status.Errorf(caErr.HTTPErrorCode(), "CSR signing error (%v)", caErr)
		}
		return nil, status.Errorf(codes.Internal, "CSR signing error (%v)", signErr)
	}
	if certSigner == "" {
		respCertChain = []string{string(cert)}
		if len(certChainBytes) != 0 {
			respCertChain = append(respCertChain, string(certChainBytes))
		}
	}
	// expand `respCertChain` since each element might be a concatenated multi-cert PEM
	response := &pb.IstioCertificateResponse{}
	for _, pem := range respCertChain {
		for _, cert := range util.PemCertBytestoString([]byte(pem)) {
			// the trailing "\n" is added for backwards compatibility
			response.CertChain = append(response.CertChain, cert+"\n")
		}
	}
	// Per the spec: "... the root cert is the last element." so we do not want to flatten the root cert.
	if len(rootCertBytes) != 0 {
		response.CertChain = append(response.CertChain, string(rootCertBytes))
	}

	klog.V(2).Infof("CSR successfully signed, sans %v.", sans)
	return response, nil
}

func (s *Server) authenticate(ctx context.Context) (*security.Caller, error) {
	if len(s.Authenticators) == 0 {
		return nil, nil
	}
	authCtx := security.AuthContext{
		GrpcContext: ctx,
	}
	for _, authenticator := range s.Authenticators {
		caller, err := authenticator.Authenticate(authCtx)
		if err == nil && caller != nil {
			return caller, nil
		}
	}
	return nil, status.Error(codes.Unauthenticated, "authentication failure")
}
