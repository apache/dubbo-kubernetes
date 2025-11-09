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

package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/features"
	securityModel "github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/security/model"
	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/cmd"
	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/pki/ca"
	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/pki/ra"
	caserver "github.com/apache/dubbo-kubernetes/dubbod/security/pkg/server/ca"
	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/server/ca/authenticate"
	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/util"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/env"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	"github.com/apache/dubbo-kubernetes/pkg/security"
	"github.com/fsnotify/fsnotify"
	"google.golang.org/grpc"
	"istio.io/api/security/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path"
	"strings"
	"time"
)

type caOptions struct {
	ExternalCAType   ra.CaExternalType
	ExternalCASigner string
	// domain to use in SPIFFE identity URLs
	TrustDomain      string
	Namespace        string
	Authenticators   []security.Authenticator
	CertSignerDomain string
}

var (
	trustedIssuer = env.Register("TOKEN_ISSUER", "",
		"OIDC token issuer. If set, will be used to check the tokens.")
	audience = env.Register("AUDIENCE", "",
		"Expected audience in the tokens. ")
	LocalCertDir = env.Register("ROOT_CA_DIR", "./etc/cacerts",
		"Location of a local or mounted CA root")
	useRemoteCerts = env.Register("USE_REMOTE_CERTS", false,
		"Whether to try to load CA certs from config Kubernetes cluster. Used for external Dubbod.")
	// TODO: Likely to be removed and added to mesh config
	externalCaType = env.Register("EXTERNAL_CA", "",
		"External CA Integration Type. Permitted value is DUBBD_RA_KUBERNETES_API.").Get()
	// TODO: Likely to be removed and added to mesh config
	k8sSigner = env.Register("K8S_SIGNER", "",
		"Kubernetes CA Signer type. Valid from Kubernetes 1.18").Get()
	k8sInCluster = env.Register("KUBERNETES_SERVICE_HOST", "",
		"Kubernetes service host, set automatically when running in-cluster")
	workloadCertTTL = env.Register("DEFAULT_WORKLOAD_CERT_TTL",
		cmd.DefaultWorkloadCertTTL,
		"The default TTL of issued workload certificates. Applied when the client sets a "+
			"non-positive TTL in the CSR.")
	maxWorkloadCertTTL = env.Register("MAX_WORKLOAD_CERT_TTL",
		cmd.DefaultMaxWorkloadCertTTL,
		"The max TTL of issued workload certificates.")
	caRSAKeySize = env.Register("AEGIS_SELF_SIGNED_CA_RSA_KEY_SIZE", 2048,
		"Specify the RSA key size to use for self-signed Dubbo CA certificates.")
	SelfSignedCACertTTL = env.Register("AEGIS_SELF_SIGNED_CA_CERT_TTL",
		cmd.DefaultSelfSignedCACertTTL,
		"The TTL of self-signed CA root certificate.")
	selfSignedRootCertGracePeriodPercentile = env.Register("AEGIS_SELF_SIGNED_ROOT_CERT_GRACE_PERIOD_PERCENTILE",
		cmd.DefaultRootCertGracePeriodPercentile,
		"Grace period percentile for self-signed root cert.")

	enableJitterForRootCertRotator = env.Register("AEGIS_ENABLE_JITTER_FOR_ROOT_CERT_ROTATOR",
		true,
		"If true, set up a jitter to start root cert rotator. "+
			"Jitter selects a backoff time in seconds to start root cert rotator, "+
			"and the back off time is below root cert check interval.")
	selfSignedRootCertCheckInterval = env.Register("AEGIS_SELF_SIGNED_ROOT_CERT_CHECK_INTERVAL",
		cmd.DefaultSelfSignedRootCertCheckInterval,
		"The interval that self-signed CA checks its root certificate "+
			"expiration time and rotates root certificate. Setting this interval "+
			"to zero or a negative value disables automated root cert check and "+
			"rotation. This interval is suggested to be larger than 10 minutes.")
)

func (s *Server) initCAServer(ca caserver.CertificateAuthority, opts *caOptions) {
	caServer, startErr := caserver.New(ca, maxWorkloadCertTTL.Get(), opts.Authenticators)
	if startErr != nil {
		log.Errorf("failed to create dubbo ca server: %v", startErr)
	}
	s.caServer = caServer
}

func (s *Server) RunCA(grpc *grpc.Server) {
	iss := trustedIssuer.Get()
	aud := audience.Get()

	token, err := os.ReadFile(securityModel.ThirdPartyJwtPath)
	if err == nil {
		tok, err := detectAuthEnv(string(token))
		if err != nil {
			log.Warnf("Starting with invalid K8S JWT token: %v", err)
		} else {
			if iss == "" {
				iss = tok.Iss
			}
			if len(tok.Aud) > 0 && len(aud) == 0 {
				aud = tok.Aud[0]
			}
		}
	}

	if iss != "" && // issuer set explicitly or extracted from our own JWT
		k8sInCluster.Get() == "" { // not running in cluster - in cluster use direct call to apiserver
		jwtRule := v1beta1.JWTRule{Issuer: iss, Audiences: []string{aud}}
		oidcAuth, err := authenticate.NewJwtAuthenticator(&jwtRule, nil)
		if err == nil {
			s.caServer.Authenticators = append(s.caServer.Authenticators, oidcAuth)
			log.Info("Using out-of-cluster JWT authentication")
		} else {
			log.Info("K8S token doesn't support OIDC, using only in-cluster auth")
		}
	}

	s.caServer.Register(grpc)

	log.Info("Dubbod CA has started")
}

func (s *Server) loadCACerts(caOpts *caOptions, dir string) error {
	if s.kubeClient == nil {
		return nil
	}

	// Skip remote fetch if a complete CA bundle is already mounted
	signingCABundleComplete, bundleExists, err := checkCABundleCompleteness(
		path.Join(dir, ca.CAPrivateKeyFile),
		path.Join(dir, ca.CACertFile),
		path.Join(dir, ca.RootCertFile),
		[]string{path.Join(dir, ca.CertChainFile)},
	)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error loading remote CA certs: %w", err)
	}
	if signingCABundleComplete {
		return nil
	}
	if bundleExists {
		log.Infof("incomplete signing CA bundle detected at %s", dir)
	}

	secret, err := s.kubeClient.Kube().CoreV1().Secrets(caOpts.Namespace).Get(
		context.TODO(), ca.CACertsSecret, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	// TODO: writing cacerts files from remote cluster will always fail,
	log.Infof("cacerts Secret found in config cluster, saving contents to %s", dir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	for key, data := range secret.Data {
		filename := path.Join(dir, key)
		if err := os.WriteFile(filename, data, 0o600); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) createDubboRA(opts *caOptions) (ra.RegistrationAuthority, error) {
	caCertFile := path.Join(ra.DefaultExtCACertDir, constants.CACertNamespaceConfigMapDataName)
	certSignerDomain := opts.CertSignerDomain
	_, err := os.Stat(caCertFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to get file info: %v", err)
		}

		// File does not exist.
		if certSignerDomain == "" {
			log.Infof("CA cert file %q not found, using %q.", caCertFile, defaultCACertPath)
			caCertFile = defaultCACertPath
		} else {
			log.Infof("CA cert file %q not found - ignoring.", caCertFile)
			caCertFile = ""
		}
	}

	if s.kubeClient == nil {
		return nil, fmt.Errorf("kubeClient is nil")
	}

	raOpts := &ra.DubboRAOptions{
		ExternalCAType:   opts.ExternalCAType,
		DefaultCertTTL:   workloadCertTTL.Get(),
		MaxCertTTL:       maxWorkloadCertTTL.Get(),
		CaSigner:         opts.ExternalCASigner,
		CaCertFile:       caCertFile,
		VerifyAppendCA:   true,
		K8sClient:        s.kubeClient.Kube(),
		TrustDomain:      opts.TrustDomain,
		CertSignerDomain: opts.CertSignerDomain,
	}
	raServer, err := ra.NewDubboRA(raOpts)
	if err != nil {
		return nil, err
	}
	raServer.SetCACertificatesFromMeshConfig(s.environment.Mesh().CaCertificates)
	s.environment.AddMeshHandler(func() {
		meshConfig := s.environment.Mesh()
		caCertificates := meshConfig.CaCertificates
		s.RA.SetCACertificatesFromMeshConfig(caCertificates)
	})
	return raServer, err
}

func (s *Server) createDubboCA(opts *caOptions) (*ca.DubboCA, error) {
	var caOpts *ca.DubboCAOptions
	var signingCABundleComplete bool
	var dubboGenerated bool
	var err error

	fileBundle, err := detectSigningCABundleAndCRL()
	if err != nil {
		return nil, fmt.Errorf("unable to determine signing file format %v", err)
	}

	signingCABundleComplete, bundleExists, err := checkCABundleCompleteness(
		fileBundle.SigningKeyFile,
		fileBundle.SigningCertFile,
		fileBundle.RootCertFile,
		fileBundle.CertChainFiles,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create an dubbod CA: %w", err)
	}
	if !signingCABundleComplete && bundleExists {
		return nil, fmt.Errorf("failed to create an dubbod CA: incomplete signing CA bundle detected")
	}

	if signingCABundleComplete {
		if _, err := os.Stat(path.Join(LocalCertDir.Get(), ca.DubboGenerated)); err == nil {
			dubboGenerated = true
		}
	}

	useSelfSignedCA := !signingCABundleComplete || (features.UseCacertsForSelfSignedCA && dubboGenerated)
	if useSelfSignedCA {
		if features.UseCacertsForSelfSignedCA && dubboGenerated {
			log.Infof("DubboGenerated %s secret found, use it as the CA certificate", ca.CACertsSecret)
		}

		// Either the secret is not mounted because it is named `dubbo-ca-secret`,
		// or it is `cacerts` secret mounted with "dubbo-generated" key set.
		caOpts, err = s.createSelfSignedCACertificateOptions(&fileBundle, opts)
		if err != nil {
			return nil, err
		}
		caOpts.OnRootCertUpdate = s.updateRootCertAndGenKeyCert
	} else {
		// The secret is mounted and the "dubbo-generated" key is not used.
		log.Info("Use local CA certificate")

		caOpts, err = ca.NewPluggedCertDubboCAOptions(fileBundle, workloadCertTTL.Get(), maxWorkloadCertTTL.Get(), caRSAKeySize.Get())
		if err != nil {
			return nil, fmt.Errorf("failed to create an dubbod CA: %v", err)
		}

		if features.EnableCACRL {
			// CRL is only supported for Plugged CA.
			// If CRL file is present, read and notify it for initial replication
			if len(fileBundle.CRLFile) > 0 {
				log.Infof("CRL file %s found, notifying it for initial replication", fileBundle.CRLFile)
				crlBytes, crlErr := os.ReadFile(fileBundle.CRLFile)
				if crlErr != nil {
					log.Errorf("failed to read CRL file %s: %v", fileBundle.CRLFile, crlErr)
					return nil, crlErr
				}

				s.dubbodCertBundleWatcher.SetAndNotifyCACRL(crlBytes)
			}
		}

		s.initCACertsAndCRLWatcher()
	}
	dubboCA, err := ca.NewDubboCA(caOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create an dubbod CA: %v", err)
	}

	// Start root cert rotator in a separate goroutine.
	dubboCA.Run(s.internalStop)
	return dubboCA, nil
}

func (s *Server) initCACertsAndCRLWatcher() {
	var err error

	s.cacertsWatcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Errorf("failed to add CAcerts watcher: %v", err)
		return
	}

	err = s.addCACertsFileWatcher(LocalCertDir.Get())
	if err != nil {
		log.Errorf("failed to add CAcerts file watcher: %v", err)
		return
	}

	go s.handleCACertsFileWatch()
}

func (s *Server) handleCACertsFileWatch() {
	var timerC <-chan time.Time
	for {
		select {
		case <-timerC:
			timerC = nil
			handleEvent(s)

		case event, ok := <-s.cacertsWatcher.Events:
			if !ok {
				log.Debugf("plugin cacerts watch stopped")
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				if timerC == nil {
					timerC = time.After(100 * time.Millisecond)
				}
			}

		case err := <-s.cacertsWatcher.Errors:
			if err != nil {
				log.Errorf("failed to catch events on cacerts file: %v", err)
				return
			}

		case <-s.internalStop:
			return
		}
	}
}

func detectAuthEnv(jwt string) (*authenticate.JwtPayload, error) {
	jwtSplit := strings.Split(jwt, ".")
	if len(jwtSplit) != 3 {
		return nil, fmt.Errorf("invalid JWT parts: %s", jwt)
	}
	payload := jwtSplit[1]

	payloadBytes, err := util.DecodeJwtPart(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode jwt: %v", err.Error())
	}

	structuredPayload := &authenticate.JwtPayload{}
	err = json.Unmarshal(payloadBytes, &structuredPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal jwt: %v", err.Error())
	}

	return structuredPayload, nil
}

func handleEvent(s *Server) {
	log.Info("Update Dubbod cacerts")

	var newCABundle []byte
	var err error
	var updateRootCA, updateCRL bool

	fileBundle, err := detectSigningCABundleAndCRL()
	if err != nil {
		log.Errorf("unable to determine signing file format %v", err)
		return
	}

	// check if CA bundle is updated
	newCABundle, err = os.ReadFile(fileBundle.RootCertFile)
	if err != nil {
		log.Errorf("failed reading root-cert.pem: %v", err)
		return
	}

	currentCABundle := s.CA.GetCAKeyCertBundle().GetRootCertPem()

	// Only updating intermediate CA is supported now
	if !bytes.Equal(currentCABundle, newCABundle) {
		if !features.MultiRootMesh {
			log.Info("Multi root is disabled, updating new ROOT-CA not supported")
			return
		}

		// in order to support root ca rotation, or we are removing the old ca,
		// we need to make the new CA bundle contain both old and new CA certs
		if bytes.Contains(currentCABundle, newCABundle) ||
			bytes.Contains(newCABundle, currentCABundle) {
			log.Info("Updating new ROOT-CA")
			updateRootCA = true
		} else {
			log.Info("Updating new ROOT-CA not supported")
			return
		}
	}

	if features.EnableCACRL {
		// check if crl file is updated
		if len(fileBundle.CRLFile) > 0 {
			currentCRLData := s.CA.GetCAKeyCertBundle().GetCRLPem()
			crlData, crlReadErr := os.ReadFile(fileBundle.CRLFile)
			if crlReadErr != nil {
				// handleEvent can be triggered either for key-cert bundle update or
				// for crl file update. So, even if there is an error in reading crl file,
				// we should log error and continue with key-cert bundle update.
				log.Errorf("failed reading crl file: %v", crlReadErr)
			}

			if !bytes.Equal(currentCRLData, crlData) {
				log.Infof("Updating CRL data")
				updateCRL = true
			}
		}
	}

	if !updateRootCA && !updateCRL {
		log.Info("No changes detected in root cert or CRL file data, skipping update")
		return
	}

	// process updated root cert or crl file
	err = s.CA.GetCAKeyCertBundle().UpdateVerifiedKeyCertBundleFromFile(
		fileBundle.SigningCertFile,
		fileBundle.SigningKeyFile,
		fileBundle.CertChainFiles,
		fileBundle.RootCertFile,
		fileBundle.CRLFile,
	)
	if err != nil {
		log.Errorf("Failed to update new Plug-in CA certs: %v", err)
		return
	}
	if len(s.CA.GetCAKeyCertBundle().GetRootCertPem()) != 0 {
		caserver.RecordCertsExpiry(s.CA.GetCAKeyCertBundle())
	}

	// notify watcher to replicate new or updated crl data
	if updateCRL {
		s.dubbodCertBundleWatcher.SetAndNotifyCACRL(s.CA.GetCAKeyCertBundle().GetCRLPem())
		log.Infof("Dubbod has detected the newly added CRL file and updated its CRL accordingly")
	}

	err = s.updateRootCertAndGenKeyCert()
	if err != nil {
		log.Errorf("Failed generating plugged-in dubbod key cert: %v", err)
		return
	}

	log.Info("Dubbod has detected the newly added intermediate CA and updated its key and certs accordingly")
}

func (s *Server) addCACertsFileWatcher(dir string) error {
	err := s.cacertsWatcher.Add(dir)
	if err != nil {
		log.Infof("failed to add cacerts file watcher for %s: %v", dir, err)
		return err
	}

	log.Infof("Added cacerts files watcher at %v", dir)

	return nil
}

func (s *Server) createSelfSignedCACertificateOptions(fileBundle *ca.SigningCAFileBundle, opts *caOptions) (*ca.DubboCAOptions, error) {
	var caOpts *ca.DubboCAOptions
	var err error
	if s.kubeClient != nil {
		log.Info("Use self-signed certificate as the CA certificate")

		// Abort after 20 minutes.
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute*20)
		defer cancel()
		// rootCertFile will be added to "ca-cert.pem".
		// readSigningCertOnly set to false - it doesn't seem to be used in AEGIS, nor do we have a way
		// to set it only for one job.
		caOpts, err = ca.NewSelfSignedDubboCAOptions(ctx,
			selfSignedRootCertGracePeriodPercentile.Get(), SelfSignedCACertTTL.Get(),
			selfSignedRootCertCheckInterval.Get(), workloadCertTTL.Get(),
			maxWorkloadCertTTL.Get(), opts.TrustDomain, features.UseCacertsForSelfSignedCA, true,
			opts.Namespace, s.kubeClient.Kube().CoreV1(), fileBundle.RootCertFile,
			enableJitterForRootCertRotator.Get(), caRSAKeySize.Get())
	} else {
		log.Infof("Use local self-signed CA certificate for testing. Will use in-memory root CA, no K8S access and no ca key file %s", fileBundle.SigningKeyFile)

		caOpts, err = ca.NewSelfSignedDebugDubboCAOptions(fileBundle.RootCertFile, SelfSignedCACertTTL.Get(),
			workloadCertTTL.Get(), maxWorkloadCertTTL.Get(), opts.TrustDomain, caRSAKeySize.Get())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create a self-signed dubbod CA: %v", err)
	}

	return caOpts, nil
}

func checkCABundleCompleteness(signingKeyFile, signingCertFile, rootCertFile string, chainFiles []string) (signingCABundleComplete bool, bundleExists bool, err error) {
	signingKeyExists, err := fileExists(signingKeyFile)
	if err != nil {
		return false, false, err
	}

	signingCertExists, err := fileExists(signingCertFile)
	if err != nil {
		return false, signingKeyExists, err
	}

	rootCertExists, err := fileExists(rootCertFile)
	if err != nil {
		return false, signingKeyExists || signingCertExists, err
	}

	chainFilesExist, err := hasValidChainFiles(chainFiles)
	if err != nil {
		return false, signingKeyExists || signingCertExists || rootCertExists, err
	}

	bundleExists = signingKeyExists || signingCertExists || rootCertExists || chainFilesExist
	signingCABundleComplete = signingKeyExists && signingCertExists && rootCertExists && chainFilesExist

	return signingCABundleComplete, bundleExists, nil
}

func detectSigningCABundleAndCRL() (ca.SigningCAFileBundle, error) {
	tlsSigningFile := path.Join(LocalCertDir.Get(), ca.TLSSecretCACertFile)

	// looking for tls file format (tls.crt)
	if _, err := os.Stat(tlsSigningFile); err == nil {
		log.Info("Using kubernetes.io/tls secret type for signing ca files")
		return ca.SigningCAFileBundle{
			RootCertFile: path.Join(LocalCertDir.Get(), ca.TLSSecretRootCertFile),
			CertChainFiles: []string{
				tlsSigningFile,
				path.Join(LocalCertDir.Get(), ca.TLSSecretRootCertFile),
			},
			SigningCertFile: tlsSigningFile,
			SigningKeyFile:  path.Join(LocalCertDir.Get(), ca.TLSSecretCAPrivateKeyFile),
		}, nil
	} else if !os.IsNotExist(err) {
		return ca.SigningCAFileBundle{}, err
	}

	log.Info("Using dubbod file format for signing ca files")
	// default ca file format
	signingCAFileBundle := ca.SigningCAFileBundle{
		RootCertFile:    path.Join(LocalCertDir.Get(), ca.RootCertFile),
		CertChainFiles:  []string{path.Join(LocalCertDir.Get(), ca.CertChainFile)},
		SigningCertFile: path.Join(LocalCertDir.Get(), ca.CACertFile),
		SigningKeyFile:  path.Join(LocalCertDir.Get(), ca.CAPrivateKeyFile),
	}

	if features.EnableCACRL {
		// load crl file if it exists
		crlFilePath := path.Join(LocalCertDir.Get(), ca.CACRLFile)
		if _, err := os.Stat(crlFilePath); err == nil {
			log.Info("Detected CRL file")
			signingCAFileBundle.CRLFile = crlFilePath
		}
	}

	return signingCAFileBundle, nil
}

func fileExists(filename string) (bool, error) {
	if filename == "" {
		return false, nil
	}
	_, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("error checking file %s: %v", filename, err)
	}
	return true, nil
}

func hasValidChainFiles(files []string) (bool, error) {
	if len(files) == 0 {
		return false, nil
	}

	for _, file := range files {
		exists, err := fileExists(file)
		if err != nil {
			return false, err
		}
		if exists {
			return true, nil
		}
	}
	return false, nil
}
