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

package security

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/grpc/peer"

	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"
)

var log = dubbolog.RegisterScope("security", "security debugging")

const (
	RootCertReqResourceName           = "ROOTCA"
	WorkloadKeyCertResourceName       = "default"
	WorkloadIdentityPath              = "./var/run/secrets/workload-spiffe-uds"
	WorkloadIdentityCredentialsPath   = "./var/run/secrets/workload-spiffe-uds/credentials"
	DefaultWorkloadIdentitySocketFile = "socket"
	DefaultCertChainFilePath          = "./etc/certs/cert-chain.pem"
	DefaultKeyFilePath                = "./etc/certs/key.pem"

	SystemRootCerts               = "SYSTEM"
	DefaultRootCertFilePath       = "./etc/certs/root-cert.pem"
	WorkloadIdentityCertChainPath = WorkloadIdentityCredentialsPath + "/cert-chain.pem"
	WorkloadIdentityRootCertPath  = WorkloadIdentityCredentialsPath + "/root-cert.pem"
	WorkloadIdentityKeyPath       = WorkloadIdentityCredentialsPath + "/key.pem"
	JWT                           = "JWT"

	CredentialMetaDataName       = "credential"
	FileRootSystemCACert         = "file-root:system"
	FileCredentialNameSocketPath = "./var/run/secrets/workload-spiffe-uds/file-credential-socket"
	CredentialNameSocketPath     = "./var/run/secrets/workload-spiffe-uds/credential-socket"
)

const (
	CertSigner = "CertSigner"
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
	ServeOnlyFiles       bool
	ProvCert             string
	FileMountedCerts     bool
	PlanetCertProvider   string
	OutputKeyCertToDir   string
	CertChainFilePath    string
	KeyFilePath          string
	RootCertFilePath     string
	CARootPath           string
	CAEndpoint           string
	CAProviderName       string
	CredFetcher          CredFetcher
	CAHeaders            map[string]string
	CAEndpointSAN        string
	CertSigner           string
	ClusterID            string
	TrustDomain          string
	STSPort              int
	CredIdentityProvider string
	XdsAuthProvider      string
	WorkloadRSAKeySize   int
	Pkcs8Keys            bool
	ECCSigAlg            string
	ECCCurve             string
	SecretTTL            time.Duration
	FileDebounceDuration time.Duration
	WorkloadNamespace    string
	ServiceAccount       string
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

func (s SdsCertificateConfig) IsRootCertificate() bool {
	return s.CaCertificatePath != ""
}

func (s SdsCertificateConfig) IsKeyCertificate() bool {
	return s.CertificatePath != "" && s.PrivateKeyPath != ""
}

const (
	ResourceSeparator = "~"
)

func SdsCertificateConfigFromResourceName(resource string) (SdsCertificateConfig, bool) {
	if strings.HasPrefix(resource, "file-cert:") {
		filesString := strings.TrimPrefix(resource, "file-cert:")
		split := strings.Split(filesString, ResourceSeparator)
		if len(split) != 2 {
			return SdsCertificateConfig{}, false
		}
		return SdsCertificateConfig{split[0], split[1], ""}, true
	} else if strings.HasPrefix(resource, "file-root:") {
		filesString := strings.TrimPrefix(resource, "file-root:")
		split := strings.Split(filesString, ResourceSeparator)

		if len(split) != 1 {
			return SdsCertificateConfig{}, false
		}
		return SdsCertificateConfig{"", "", split[0]}, true
	}
	return SdsCertificateConfig{}, false
}

func GetWorkloadSDSSocketListenPath(sockfile string) string {
	return filepath.Join(WorkloadIdentityPath, sockfile)
}

func GetDubboSDSServerSocketPath() string {
	return filepath.Join(WorkloadIdentityPath, DefaultWorkloadIdentitySocketFile)
}

func GetOSRootFilePath() string {
	// Get and store the OS CA certificate path for Linux systems
	// Source of CA File Paths: https://golang.org/src/crypto/x509/root_linux.go
	certFiles := []string{
		"/etc/ssl/certs/ca-certificates.crt",                // Debian/Ubuntu/Gentoo etc.
		"/etc/pki/tls/certs/ca-bundle.crt",                  // Fedora/RHEL 6
		"/etc/ssl/ca-bundle.pem",                            // OpenSUSE
		"/etc/pki/tls/cacert.pem",                           // OpenELEC
		"/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem", // CentOS/RHEL 7
		"/etc/ssl/cert.pem",                                 // Alpine Linux
		"/usr/local/etc/ssl/cert.pem",                       // FreeBSD
		"/etc/ssl/certs/ca-certificates",                    // Talos Linux
	}

	for _, cert := range certFiles {
		if _, err := os.Stat(cert); err == nil {
			log.Infof("Using OS CA certificate for proxy: %s", cert)
			return cert
		}
	}
	log.Info("OS CA Cert could not be found for agent")
	return ""
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

// GetConnectionAddress extracts the peer address from the gRPC context.
// It returns "unknown" if the peer information is not available.
func GetConnectionAddress(ctx context.Context) string {
	peerInfo, ok := peer.FromContext(ctx)
	peerAddr := "unknown"
	if ok {
		peerAddr = peerInfo.Addr.String()
	}
	return peerAddr
}
