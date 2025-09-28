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

package ra

import (
	"bytes"
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/slices"
	"github.com/apache/dubbo-kubernetes/security/pkg/pki/ca"
	raerror "github.com/apache/dubbo-kubernetes/security/pkg/pki/error"
	"github.com/apache/dubbo-kubernetes/security/pkg/pki/util"
	caserver "github.com/apache/dubbo-kubernetes/security/pkg/server/ca"
	meshconfig "istio.io/api/mesh/v1alpha1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"strings"
	"time"
)

type CaExternalType string

const (
	ExtCAK8s            CaExternalType = "DUBBOD_RA_KUBERNETES_API"
	DefaultExtCACertDir string         = "./etc/external-ca-cert"
)

type RegistrationAuthority interface {
	caserver.CertificateAuthority
	SetCACertificatesFromMeshConfig([]*meshconfig.MeshConfig_CertificateData)
	GetRootCertFromMeshConfig(signerName string) ([]byte, error)
}

type DubboRAOptions struct {
	ExternalCAType   CaExternalType
	DefaultCertTTL   time.Duration
	MaxCertTTL       time.Duration
	CaCertFile       string
	CaSigner         string
	VerifyAppendCA   bool
	K8sClient        clientset.Interface
	TrustDomain      string
	CertSignerDomain string
}

// NewDubboRA is a factory method that returns an RA that implements the RegistrationAuthority functionality.
// the caOptions defines the external provider
func NewDubboRA(opts *DubboRAOptions) (RegistrationAuthority, error) {
	if opts.ExternalCAType == ExtCAK8s {
		dubboRA, err := NewKubernetesRA(opts)
		if err != nil {
			return nil, fmt.Errorf("failed to create an K8s CA: %v", err)
		}
		return dubboRA, err
	}
	return nil, fmt.Errorf("invalid CA Name %s", opts.ExternalCAType)
}

func (r *KubernetesRA) SignWithCertChain(csrPEM []byte, certOpts ca.CertOpts) ([]string, error) {
	cert, err := r.Sign(csrPEM, certOpts)
	if err != nil {
		return nil, err
	}
	chainPem := r.GetCAKeyCertBundle().GetCertChainPem()
	if len(chainPem) > 0 {
		cert = append(cert, chainPem...)
	}
	respCertChain := []string{string(cert)}
	var possibleRootCert, rootCertFromMeshConfig, rootCertFromCertChain []byte
	certSigner := r.certSignerDomain + "/" + certOpts.CertSigner
	if len(r.GetCAKeyCertBundle().GetRootCertPem()) == 0 {
		rootCertFromCertChain, err = util.FindRootCertFromCertificateChainBytes(cert)
		if err != nil {
			klog.Infof("failed to find root cert from signed cert-chain (%v)", err.Error())
		}
		rootCertFromMeshConfig, err = r.GetRootCertFromMeshConfig(certSigner)
		if err != nil {
			klog.Infof("failed to find root cert from mesh config (%v)", err.Error())
		}
		if rootCertFromMeshConfig != nil {
			possibleRootCert = rootCertFromMeshConfig
		} else if rootCertFromCertChain != nil {
			possibleRootCert = rootCertFromCertChain
		}
		if possibleRootCert == nil {
			return nil, raerror.NewError(raerror.CSRError, fmt.Errorf("failed to find root cert from either signed cert-chain or mesh config"))
		}
		if verifyErr := util.VerifyCertificate(nil, cert, possibleRootCert, nil); verifyErr != nil {
			return nil, raerror.NewError(raerror.CSRError, fmt.Errorf("root cert from signed cert-chain is invalid (%v)", verifyErr))
		}
		if !bytes.Equal(possibleRootCert, rootCertFromCertChain) {
			respCertChain = append(respCertChain, string(possibleRootCert))
		}
	}
	return respCertChain, nil
}

// GetCAKeyCertBundle returns the KeyCertBundle for the CA.
func (r *KubernetesRA) GetCAKeyCertBundle() *util.KeyCertBundle {
	return r.keyCertBundle
}

func (r *KubernetesRA) SetCACertificatesFromMeshConfig(caCertificates []*meshconfig.MeshConfig_CertificateData) {
	r.mutex.Lock()
	for _, pemCert := range caCertificates {
		// TODO:  take care of spiffe bundle format as well
		cert := pemCert.GetPem()
		certSigners := pemCert.CertSigners
		if len(certSigners) != 0 {
			certSigner := strings.Join(certSigners, ",")
			if cert != "" {
				r.caCertificatesFromMeshConfig[certSigner] = cert
			}
		}
	}
	r.mutex.Unlock()
}

func (r *KubernetesRA) GetRootCertFromMeshConfig(signerName string) ([]byte, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	caCertificates := r.caCertificatesFromMeshConfig
	if len(caCertificates) == 0 {
		return nil, fmt.Errorf("no caCertificates defined in mesh config")
	}
	for signers, caCertificate := range caCertificates {
		signerList := strings.Split(signers, ",")
		if len(signerList) == 0 {
			continue
		}
		for _, signer := range signerList {
			if signer == signerName {
				return []byte(caCertificate), nil
			}
		}
	}
	return nil, fmt.Errorf("failed to find root cert for signer: %v in mesh config", signerName)
}

func ValidateCSR(csrPEM []byte, subjectIDs []string) bool {
	csr, err := util.ParsePemEncodedCSR(csrPEM)
	if err != nil {
		return false
	}
	if err := csr.CheckSignature(); err != nil {
		return false
	}
	csrIDs, err := util.ExtractIDs(csr.Extensions)
	if err != nil {
		return false
	}
	for _, s1 := range csrIDs {
		if !slices.Contains(subjectIDs, s1) {
			return false
		}
	}

	hosts := strings.Join(csrIDs, ",")
	genCSRTemplate, err := util.GenCSRTemplate(util.CertOptions{Host: hosts})
	if err != nil {
		return false
	}
	if len(csr.Subject.Organization) == 0 {
		csr.Subject.Organization = []string{""}
	}
	if !compareCSRs(csr, genCSRTemplate) {
		return false
	}
	return true
}

func compareCSRs(orgCSR, genCSR *x509.CertificateRequest) bool {
	// Compare the CSR fields
	if orgCSR == nil || genCSR == nil {
		return false
	}

	orgSubj, err := asn1.Marshal(orgCSR.Subject.ToRDNSequence())
	if err != nil {
		return false
	}
	gensubj, err := asn1.Marshal(genCSR.Subject.ToRDNSequence())
	if err != nil {
		return false
	}

	if !bytes.Equal(orgSubj, gensubj) {
		return false
	}
	// Expected length is 0 or 1
	if len(orgCSR.URIs) > 1 {
		return false
	}
	// Expected length is 0
	if len(orgCSR.EmailAddresses) != len(genCSR.EmailAddresses) {
		return false
	}
	// Expexted length is 0
	if len(orgCSR.IPAddresses) != len(genCSR.IPAddresses) {
		return false
	}
	// Expected length is 0
	if len(orgCSR.DNSNames) != len(genCSR.DNSNames) {
		return false
	}
	// Only SAN extensions are expected in the orgCSR
	for _, extension := range orgCSR.Extensions {
		switch {
		case extension.Id.Equal(util.OidSubjectAlternativeName):
		default:
			return false // no other extension used.
		}
	}
	// ExtraExtensions should not be populated in the orgCSR
	return len(orgCSR.ExtraExtensions) == 0
}

func preSign(raOpts *DubboRAOptions, csrPEM []byte, subjectIDs []string, requestedLifetime time.Duration, forCA bool) (time.Duration, error) {
	if forCA {
		return requestedLifetime, raerror.NewError(raerror.CSRError,
			fmt.Errorf("unable to generate CA certifificates"))
	}
	if !ValidateCSR(csrPEM, subjectIDs) {
		return requestedLifetime, raerror.NewError(raerror.CSRError, fmt.Errorf(
			"unable to validate SAN Identities in CSR"))
	}
	// If the requested requestedLifetime is non-positive, apply the default TTL.
	lifetime := requestedLifetime
	if requestedLifetime.Seconds() <= 0 {
		lifetime = raOpts.DefaultCertTTL
	}
	// If the requested TTL is greater than maxCertTTL, return an error
	if requestedLifetime.Seconds() > raOpts.MaxCertTTL.Seconds() {
		return lifetime, raerror.NewError(raerror.TTLError, fmt.Errorf(
			"requested TTL %s is greater than the max allowed TTL %s", requestedLifetime, raOpts.MaxCertTTL))
	}
	return lifetime, nil
}
