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

package ra

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/k8s/chiron"
	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/pki/ca"
	raerror "github.com/apache/dubbo-kubernetes/dubbod/security/pkg/pki/error"
	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/pki/util"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	cert "k8s.io/api/certificates/v1"
	clientset "k8s.io/client-go/kubernetes"
	"sync"
	"time"
)

var pkiRaLog = log.RegisterScope("pkira", "Dubbod RA log")

// KubernetesRA integrated with an external CA using Kubernetes CSR API
type KubernetesRA struct {
	csrInterface                       clientset.Interface
	keyCertBundle                      *util.KeyCertBundle
	raOpts                             *DubboRAOptions
	caCertificatesFromMeshGlobalConfig map[string]string
	certSignerDomain                   string
	// mutex protects the R/W to caCertificatesFromMeshGlobalConfig.
	mutex sync.RWMutex
}

func NewKubernetesRA(raOpts *DubboRAOptions) (*KubernetesRA, error) {
	keyCertBundle, err := util.NewKeyCertBundleWithRootCertFromFile(raOpts.CaCertFile)
	if err != nil {
		return nil, raerror.NewError(raerror.CAInitFail, fmt.Errorf("error processing Certificate Bundle for Kubernetes RA"))
	}
	dubboRA := &KubernetesRA{
		csrInterface:                       raOpts.K8sClient,
		raOpts:                             raOpts,
		keyCertBundle:                      keyCertBundle,
		certSignerDomain:                   raOpts.CertSignerDomain,
		caCertificatesFromMeshGlobalConfig: make(map[string]string),
	}
	return dubboRA, nil
}

// Sign takes a PEM-encoded CSR and cert opts, and returns a certificate signed by k8s CA.
func (r *KubernetesRA) Sign(csrPEM []byte, certOpts ca.CertOpts) ([]byte, error) {
	_, err := preSign(r.raOpts, csrPEM, certOpts.SubjectIDs, certOpts.TTL, certOpts.ForCA)
	if err != nil {
		return nil, err
	}
	certSigner := certOpts.CertSigner

	return r.kubernetesSign(csrPEM, r.raOpts.CaCertFile, certSigner, certOpts.TTL)
}

func (r *KubernetesRA) kubernetesSign(csrPEM []byte, caCertFile string, certSigner string, requestedLifetime time.Duration) ([]byte, error) {
	certSignerDomain := r.certSignerDomain
	if certSignerDomain == "" && certSigner != "" {
		return nil, raerror.NewError(raerror.CertGenError, fmt.Errorf("certSignerDomain is required for signer %s", certSigner))
	}
	if certSignerDomain != "" && certSigner != "" {
		certSigner = certSignerDomain + "/" + certSigner
	} else {
		certSigner = r.raOpts.CaSigner
	}
	usages := []cert.KeyUsage{
		cert.UsageDigitalSignature,
		cert.UsageKeyEncipherment,
		cert.UsageServerAuth,
		cert.UsageClientAuth,
	}
	certChain, _, err := chiron.SignCSRK8s(r.csrInterface, csrPEM, certSigner, usages, "", caCertFile, true, false, requestedLifetime)
	if err != nil {
		return nil, raerror.NewError(raerror.CertGenError, err)
	}
	return certChain, err
}
