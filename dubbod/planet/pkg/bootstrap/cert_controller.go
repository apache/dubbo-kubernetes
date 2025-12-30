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
	"bytes"
	"fmt"
	tb "github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/trustbundle"
	"os"
	"path"
	"strings"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/features"
	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/k8s/chiron"
	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/pki/ca"
	certutil "github.com/apache/dubbo-kubernetes/dubbod/security/pkg/util"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	"github.com/apache/dubbo-kubernetes/pkg/sleep"
)

const (
	defaultCertGracePeriodRatio = 0.5
	rootCertPollingInterval     = 60 * time.Second
	defaultCACertPath           = "./var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
)

func (s *Server) initFileCertificateWatches(tlsOptions TLSOptions) error {
	if err := s.dubbodCertBundleWatcher.SetFromFilesAndNotify(tlsOptions.KeyFile, tlsOptions.CertFile, tlsOptions.CaCertFile); err != nil {
		return fmt.Errorf("set keyCertBundle failed: %v", err)
	}
	// TODO: Setup watcher for root and restart server if it changes.
	for _, file := range []string{tlsOptions.CertFile, tlsOptions.KeyFile} {
		log.Infof("adding watcher for certificate %s", file)
		if err := s.fileWatcher.Add(file); err != nil {
			return fmt.Errorf("could not watch %v: %v", file, err)
		}
	}
	s.addStartFunc("certificate rotation", func(stop <-chan struct{}) error {
		go func() {
			var keyCertTimerC <-chan time.Time
			for {
				select {
				case <-keyCertTimerC:
					keyCertTimerC = nil
					if err := s.dubbodCertBundleWatcher.SetFromFilesAndNotify(tlsOptions.KeyFile, tlsOptions.CertFile, tlsOptions.CaCertFile); err != nil {
						log.Errorf("Setting keyCertBundle failed: %v", err)
					}
				case <-s.fileWatcher.Events(tlsOptions.CertFile):
					if keyCertTimerC == nil {
						keyCertTimerC = time.After(watchDebounceDelay)
					}
				case <-s.fileWatcher.Events(tlsOptions.KeyFile):
					if keyCertTimerC == nil {
						keyCertTimerC = time.After(watchDebounceDelay)
					}
				case err := <-s.fileWatcher.Errors(tlsOptions.CertFile):
					log.Errorf("error watching %v: %v", tlsOptions.CertFile, err)
				case err := <-s.fileWatcher.Errors(tlsOptions.KeyFile):
					log.Errorf("error watching %v: %v", tlsOptions.KeyFile, err)
				case <-stop:
					return
				}
			}
		}()
		return nil
	})
	return nil
}

func (s *Server) RotateDNSCertForK8sCA(stop <-chan struct{}, defaultCACertPath string, signerName string, approveCsr bool, requestedLifetime time.Duration) {
	certUtil := certutil.NewCertUtil(int(defaultCertGracePeriodRatio * 100))
	for {
		waitTime, _ := certUtil.GetWaitTime(s.dubbodCertBundleWatcher.GetKeyCertBundle().CertPem, time.Now())
		if !sleep.Until(stop, waitTime) {
			return
		}
		certChain, keyPEM, _, err := chiron.GenKeyCertK8sCA(s.kubeClient.Kube(),
			strings.Join(s.dnsNames, ","), defaultCACertPath, signerName, approveCsr, requestedLifetime)
		if err != nil {
			log.Errorf("failed regenerating key and cert for dubbod by kubernetes: %v", err)
			continue
		}
		s.dubbodCertBundleWatcher.SetAndNotify(keyPEM, certChain, s.dubbodCertBundleWatcher.GetCABundle())
	}
}

func (s *Server) initDNSCertsK8SRA() error {
	var certChain, keyPEM, caBundle []byte
	var err error
	planetCertProviderName := features.PlanetCertProvider

	signerName := strings.TrimPrefix(planetCertProviderName, constants.CertProviderKubernetesSignerPrefix)
	log.Infof("Generating K8S-signed cert for %v using signer %v", s.dnsNames, signerName)
	certChain, keyPEM, _, err = chiron.GenKeyCertK8sCA(s.kubeClient.Kube(),
		strings.Join(s.dnsNames, ","), "", signerName, true, SelfSignedCACertTTL.Get())
	if err != nil {
		return fmt.Errorf("failed generating key and cert by kubernetes: %v", err)
	}
	caBundle, err = s.RA.GetRootCertFromMeshGlobalConfig(signerName)
	if err != nil {
		return err
	}

	// MeshGlobalConfig:Add callback for mesh global config update
	s.environment.AddMeshHandler(func() {
		newCaBundle, _ := s.RA.GetRootCertFromMeshGlobalConfig(signerName)
		if newCaBundle != nil && !bytes.Equal(newCaBundle, s.dubbodCertBundleWatcher.GetKeyCertBundle().CABundle) {
			newCertChain, newKeyPEM, _, err := chiron.GenKeyCertK8sCA(s.kubeClient.Kube(),
				strings.Join(s.dnsNames, ","), "", signerName, true, SelfSignedCACertTTL.Get())
			if err != nil {
				log.Errorf("failed regenerating key and cert for dubbod by kubernetes: %v", err)
			}
			s.dubbodCertBundleWatcher.SetAndNotify(newKeyPEM, newCertChain, newCaBundle)
		}
	})

	s.addStartFunc("dubbod server certificate rotation", func(stop <-chan struct{}) error {
		go func() {
			// Track TTL of DNS cert and renew cert in accordance to grace period.
			s.RotateDNSCertForK8sCA(stop, "", signerName, true, SelfSignedCACertTTL.Get())
		}()
		return nil
	})
	s.dubbodCertBundleWatcher.SetAndNotify(keyPEM, certChain, caBundle)
	return nil
}

func (s *Server) initDNSCertsDubbod() error {
	var certChain, keyPEM, caBundle []byte
	var err error
	// Generate certificates for Dubbod DNS names, signed by Dubbod CA
	certChain, keyPEM, err = s.CA.GenKeyCert(s.dnsNames, SelfSignedCACertTTL.Get(), false)
	if err != nil {
		return fmt.Errorf("failed generating dubbod key cert %v", err)
	}
	log.Infof("Generating dubbod-signed cert for %v:\n %s", s.dnsNames, certChain)

	fileBundle, err := detectSigningCABundleAndCRL()
	if err != nil {
		return fmt.Errorf("unable to determine signing file format %v", err)
	}

	dubboGenerated, detectedSigningCABundle := false, false
	if _, err := os.Stat(fileBundle.SigningKeyFile); err == nil {
		detectedSigningCABundle = true
		if _, err := os.Stat(path.Join(LocalCertDir.Get(), ca.DubboGenerated)); err == nil {
			dubboGenerated = true
		}
	}

	// check if signing key file exists the cert dir and if the dubbo-generated file
	// exists (only if USE_CACERTS_FOR_SELF_SIGNED_CA is enabled)
	if !detectedSigningCABundle {
		log.Infof("Use roots from dubbo-ca-secret")

		caBundle = s.CA.GetCAKeyCertBundle().GetRootCertPem()
		s.addStartFunc("dubbod server certificate rotation", func(stop <-chan struct{}) error {
			go func() {
				// regenerate dubbod key cert when root cert changes.
				s.watchRootCertAndGenKeyCert(stop)
			}()
			return nil
		})
	} else if features.UseCacertsForSelfSignedCA && dubboGenerated {
		log.Infof("Use roots from %v and watch", fileBundle.RootCertFile)

		caBundle = s.CA.GetCAKeyCertBundle().GetRootCertPem()
		// Similar code to dubbo-ca-secret: refresh the root cert, but in casecrets
		s.addStartFunc("dubbod server certificate rotation", func(stop <-chan struct{}) error {
			go func() {
				// regenerate dubbod key cert when root cert changes.
				s.watchRootCertAndGenKeyCert(stop)
			}()
			return nil
		})

	} else {
		log.Infof("Use root cert from %v", fileBundle.RootCertFile)

		caBundle, err = os.ReadFile(fileBundle.RootCertFile)
		if err != nil {
			return fmt.Errorf("failed reading %s: %v", fileBundle.RootCertFile, err)
		}
	}
	s.dubbodCertBundleWatcher.SetAndNotify(keyPEM, certChain, caBundle)
	return nil
}

func (s *Server) watchRootCertAndGenKeyCert(stop <-chan struct{}) {
	caBundle := s.CA.GetCAKeyCertBundle().GetRootCertPem()
	for {
		if !sleep.Until(stop, rootCertPollingInterval) {
			return
		}
		newRootCert := s.CA.GetCAKeyCertBundle().GetRootCertPem()
		if !bytes.Equal(caBundle, newRootCert) {
			caBundle = newRootCert
			certChain, keyPEM, err := s.CA.GenKeyCert(s.dnsNames, SelfSignedCACertTTL.Get(), false)
			if err != nil {
				log.Errorf("failed generating dubbod key cert %v", err)
			} else {
				s.dubbodCertBundleWatcher.SetAndNotify(keyPEM, certChain, caBundle)
				log.Infof("regenerated dubbod dns cert: %s", certChain)
			}
		}
	}
}

func (s *Server) updateRootCertAndGenKeyCert() error {
	log.Infof("update root cert and generate new dns certs")
	caBundle := s.CA.GetCAKeyCertBundle().GetRootCertPem()
	certChain, keyPEM, err := s.CA.GenKeyCert(s.dnsNames, SelfSignedCACertTTL.Get(), false)
	if err != nil {
		return err
	}

	if features.MultiRootMesh {
		// Trigger trust anchor update, this will send PCDS to all sidecars.
		log.Infof("Update trust anchor with new root cert")
		err = s.workloadTrustBundle.UpdateTrustAnchor(&tb.TrustAnchorUpdate{
			TrustAnchorConfig: tb.TrustAnchorConfig{Certs: []string{string(caBundle)}},
			Source:            tb.SourceDubboCA,
		})
		if err != nil {
			log.Errorf("failed to update trust anchor from source Dubbo CA, err: %v", err)
			return err
		}
	}

	s.dubbodCertBundleWatcher.SetAndNotify(keyPEM, certChain, caBundle)
	return nil
}
