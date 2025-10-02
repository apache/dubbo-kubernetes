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

package security

import (
	"context"
	"fmt"
	"google.golang.org/grpc/metadata"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	RootCertReqResourceName           = "ROOTCA"
	WorkloadKeyCertResourceName       = "default"
	WorkloadIdentityPath              = "./var/run/secrets/workload-spiffe-uds"
	DefaultWorkloadIdentitySocketFile = "socket"
	SystemRootCerts                   = "SYSTEM"
	DefaultRootCertFilePath           = "./etc/certs/root-cert.pem"
	CredentialNameSocketPath          = "./var/run/secrets/credential-uds/socket"
	WorkloadIdentityCredentialsPath   = "./var/run/secrets/workload-spiffe-credentials"
	WorkloadIdentityCertChainPath     = WorkloadIdentityCredentialsPath + "/cert-chain.pem"
	WorkloadIdentityRootCertPath      = WorkloadIdentityCredentialsPath + "/root-cert.pem"
	WorkloadIdentityKeyPath           = WorkloadIdentityCredentialsPath + "/key.pem"
	FileCredentialNameSocketPath      = "./var/run/secrets/credential-uds/files-socket"
	JWT                               = "JWT"
)

const (
	BearerTokenPrefix = "Bearer "
	K8sTokenPrefix    = "Dubbo "
	CertSigner        = "CertSigner"
)

type AuthContext struct {
	GrpcContext context.Context
	Request     *http.Request
}

type Authenticator interface {
	Authenticate(ctx AuthContext) (*Caller, error)
	AuthenticatorType() string
}

type SecretItem struct {
	CertificateChain []byte
	PrivateKey       []byte
	RootCert         []byte
	ResourceName     string
	CreatedTime      time.Time
	ExpireTime       time.Time
}

type SecretManager interface {
	GenerateSecret(resourceName string) (*SecretItem, error)
}

type SdsCertificateConfig struct {
	CertificatePath   string
	PrivateKeyPath    string
	CaCertificatePath string
}

type AuthSource int

const (
	AuthSourceClientCertificate AuthSource = iota
	AuthSourceIDToken
)

const (
	authorizationMeta = "authorization"
)

type KubernetesInfo struct {
	PodName           string
	PodNamespace      string
	PodUID            string
	PodServiceAccount string
}

type Caller struct {
	AuthSource AuthSource
	Identities []string

	KubernetesInfo KubernetesInfo
}

type Options struct {
	ServeOnlyFiles     bool
	ProvCert           string
	FileMountedCerts   bool
	SailCertProvider   string
	OutputKeyCertToDir string
	CertChainFilePath  string
	KeyFilePath        string
	RootCertFilePath   string
	CARootPath         string
	CAEndpoint         string
	CAProviderName     string
	CredFetcher        CredFetcher
	CAHeaders          map[string]string
	CAEndpointSAN      string
	CertSigner         string
	ClusterID          string
	TrustDomain        string
}

type CredFetcher interface {
	GetPlatformCredential() (string, error)
	GetIdentityProvider() string
	Stop()
}

type Client interface {
	CSRSign(csrPEM []byte, certValidTTLInSec int64) ([]string, error)
	Close()
	GetRootCertBundle() ([]string, error)
}

func GetWorkloadSDSSocketListenPath(sockfile string) string {
	return filepath.Join(WorkloadIdentityPath, sockfile)
}

func GetDubboSDSServerSocketPath() string {
	return filepath.Join(WorkloadIdentityPath, DefaultWorkloadIdentitySocketFile)
}

func CheckWorkloadCertificate(certChainFilePath, keyFilePath, rootCertFilePath string) bool {
	if _, err := os.Stat(certChainFilePath); err != nil {
		return false
	}
	if _, err := os.Stat(keyFilePath); err != nil {
		return false
	}
	if _, err := os.Stat(rootCertFilePath); err != nil {
		return false
	}
	return true
}

func ExtractBearerToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", fmt.Errorf("no metadata is attached")
	}

	authHeader, exists := md[authorizationMeta]
	if !exists {
		return "", fmt.Errorf("no HTTP authorization header exists")
	}

	for _, value := range authHeader {
		if strings.HasPrefix(value, BearerTokenPrefix) {
			return strings.TrimPrefix(value, BearerTokenPrefix), nil
		}
	}

	return "", fmt.Errorf("no bearer token exists in HTTP authorization header")
}

func ExtractRequestToken(req *http.Request) (string, error) {
	value := req.Header.Get(authorizationMeta)
	if value == "" {
		return "", fmt.Errorf("no HTTP authorization header exists")
	}

	if strings.HasPrefix(value, BearerTokenPrefix) {
		return strings.TrimPrefix(value, BearerTokenPrefix), nil
	}
	if strings.HasPrefix(value, K8sTokenPrefix) {
		return strings.TrimPrefix(value, K8sTokenPrefix), nil
	}

	return "", fmt.Errorf("no bearer token exists in HTTP authorization header")
}
