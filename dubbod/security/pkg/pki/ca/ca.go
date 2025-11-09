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
	"crypto/elliptic"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/cmd"
	caerror "github.com/apache/dubbo-kubernetes/dubbod/security/pkg/pki/error"
	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/pki/util"
	certutil "github.com/apache/dubbo-kubernetes/dubbod/security/pkg/util"
	"github.com/apache/dubbo-kubernetes/pkg/backoff"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	v1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var pkiCaLog = log.RegisterScope("pkica", "Aegis CA log")

const (
	CACertFile                = "ca-cert.pem"
	CAPrivateKeyFile          = "ca-key.pem"
	CACRLFile                 = "ca-crl.pem"
	CertChainFile             = "cert-chain.pem"
	PrivateKeyFile            = "key.pem"
	RootCertFile              = "root-cert.pem"
	TLSSecretCACertFile       = "tls.crt"
	TLSSecretCAPrivateKeyFile = "tls.key"
	TLSSecretRootCertFile     = "ca.crt"
	rsaKeySize                = 2048
	CACertsSecret             = "cacerts"
	CASecret                  = "dubbo-ca-secret"
	DubboGenerated            = "dubbo-generated"
)

var (
	dubboCASecretType = v1.SecretTypeOpaque
)

const (
	// selfSignedCA means the Dubbo CA uses a self signed certificate.
	selfSignedCA caTypes = iota
	// pluggedCertCA means the Dubbo CA uses a operator-specified key/cert.
	pluggedCertCA
)

type SigningCAFileBundle struct {
	RootCertFile    string
	CertChainFiles  []string
	SigningCertFile string
	SigningKeyFile  string
	CRLFile         string
}

type caTypes int

type CertOpts struct {
	SubjectIDs []string
	TTL        time.Duration
	ForCA      bool
	CertSigner string
}

type DubboCA struct {
	defaultCertTTL  time.Duration
	maxCertTTL      time.Duration
	caRSAKeySize    int
	keyCertBundle   *util.KeyCertBundle
	rootCertRotator *SelfSignedCARootCertRotator
}

type DubboCAOptions struct {
	CAType caTypes

	DefaultCertTTL time.Duration
	MaxCertTTL     time.Duration
	CARSAKeySize   int

	KeyCertBundle *util.KeyCertBundle

	// Config for creating self-signed root cert rotator.
	RotatorConfig *SelfSignedCARootCertRotatorConfig

	// OnRootCertUpdate is the cb which can only be called by self-signed root cert rotator
	OnRootCertUpdate func() error
}

func NewDubboCA(opts *DubboCAOptions) (*DubboCA, error) {
	ca := &DubboCA{
		maxCertTTL:    opts.MaxCertTTL,
		keyCertBundle: opts.KeyCertBundle,
		caRSAKeySize:  opts.CARSAKeySize,
	}

	if opts.CAType == selfSignedCA && opts.RotatorConfig != nil && opts.RotatorConfig.CheckInterval > time.Duration(0) {
		ca.rootCertRotator = NewSelfSignedCARootCertRotator(opts.RotatorConfig, ca, opts.OnRootCertUpdate)
	}

	// if CA cert becomes invalid before workload cert it's going to cause workload cert to be invalid too,
	// however citatel won't rotate if that happens, this function will prevent that using cert chain TTL as
	// the workload TTL
	defaultCertTTL, err := ca.minTTL(opts.DefaultCertTTL)
	if err != nil {
		return ca, fmt.Errorf("failed to get default cert TTL %s", err.Error())
	}
	ca.defaultCertTTL = defaultCertTTL

	return ca, nil
}

func NewSelfSignedDubboCAOptions(ctx context.Context,
	rootCertGracePeriodPercentile int, caCertTTL, rootCertCheckInverval, defaultCertTTL,
	maxCertTTL time.Duration, org string, useCacertsSecretName, dualUse bool, namespace string, client corev1.CoreV1Interface,
	rootCertFile string, enableJitter bool, caRSAKeySize int,
) (caOpts *DubboCAOptions, err error) {
	caOpts = &DubboCAOptions{
		CAType:         selfSignedCA,
		DefaultCertTTL: defaultCertTTL,
		MaxCertTTL:     maxCertTTL,
		RotatorConfig: &SelfSignedCARootCertRotatorConfig{
			CheckInterval:      rootCertCheckInverval,
			caCertTTL:          caCertTTL,
			retryInterval:      cmd.ReadSigningCertRetryInterval,
			retryMax:           cmd.ReadSigningCertRetryMax,
			certInspector:      certutil.NewCertUtil(rootCertGracePeriodPercentile),
			caStorageNamespace: namespace,
			dualUse:            dualUse,
			org:                org,
			rootCertFile:       rootCertFile,
			enableJitter:       enableJitter,
			client:             client,
		},
	}

	// always use ``dubbo-ca-secret` in priority, otherwise fall back to `cacerts`
	var caCertName string
	b := backoff.NewExponentialBackOff(backoff.DefaultOption())
	err = b.RetryWithContext(ctx, func() error {
		caCertName = CASecret
		// 1. fetch `dubbo-ca-secret` in priority
		err := loadSelfSignedCaSecret(client, namespace, caCertName, rootCertFile, caOpts)
		if err == nil {
			return nil
		} else if apierror.IsNotFound(err) {
			// 2. if `dubbo-ca-secret` not exist and use cacerts enabled, fallback to fetch `cacerts`
			if useCacertsSecretName {
				caCertName = CACertsSecret
				err := loadSelfSignedCaSecret(client, namespace, caCertName, rootCertFile, caOpts)
				if err == nil {
					return nil
				} else if apierror.IsNotFound(err) { // if neither `dubbo-ca-secret` nor `cacerts` exists, we create a `cacerts`
					// continue to create `cacerts`
				} else {
					return err
				}
			}

			// 3. if use cacerts disabled, create `dubbo-ca-secret`, otherwise create `cacerts`.
			pkiCaLog.Infof("CASecret %s not found, will create one", caCertName)
			options := util.CertOptions{
				TTL:          caCertTTL,
				Org:          org,
				IsCA:         true,
				IsSelfSigned: true,
				RSAKeySize:   caRSAKeySize,
				IsDualUse:    dualUse,
			}
			pemCert, pemKey, ckErr := util.GenCertKeyFromOptions(options)
			if ckErr != nil {
				pkiCaLog.Errorf("unable to generate CA cert and key for self-signed CA (%v)", ckErr)
				return fmt.Errorf("unable to generate CA cert and key for self-signed CA (%v)", ckErr)
			}

			rootCerts, err := util.AppendRootCerts(pemCert, rootCertFile)
			if err != nil {
				pkiCaLog.Errorf("failed to append root certificates (%v)", err)
				return fmt.Errorf("failed to append root certificates (%v)", err)
			}
			if caOpts.KeyCertBundle, err = util.NewVerifiedKeyCertBundleFromPem(
				pemCert,
				pemKey,
				nil,
				rootCerts,
				nil,
			); err != nil {
				pkiCaLog.Errorf("failed to create CA KeyCertBundle (%v)", err)
				return fmt.Errorf("failed to create CA KeyCertBundle (%v)", err)
			}
			// Write the key/cert back to secret, so they will be persistent when CA restarts.
			secret := BuildSecret(caCertName, namespace, nil, nil, pemCert, pemCert, pemKey, dubboCASecretType)
			_, err = client.Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
			if err != nil {
				pkiCaLog.Errorf("Failed to create secret %s (%v)", caCertName, err)
				return err
			}
			pkiCaLog.Infof("Using self-generated public key: %v", string(rootCerts))
			return nil
		}
		return err
	})
	pkiCaLog.Infof("Set secret name for self-signed CA cert rotator to %s", caCertName)
	caOpts.RotatorConfig.secretName = caCertName
	return caOpts, err
}

func (ca *DubboCA) Run(stopChan chan struct{}) {
	if ca.rootCertRotator != nil {
		// Start root cert rotator in a separate goroutine.
		go ca.rootCertRotator.Run(stopChan)
	}
}

func (ca *DubboCA) Sign(csrPEM []byte, certOpts CertOpts) ([]byte, error) {
	return ca.sign(csrPEM, certOpts.SubjectIDs, certOpts.TTL, true, certOpts.ForCA)
}

func (ca *DubboCA) SignWithCertChain(csrPEM []byte, certOpts CertOpts) ([]string, error) {
	cert, err := ca.signWithCertChain(csrPEM, certOpts.SubjectIDs, certOpts.TTL, true, certOpts.ForCA)
	if err != nil {
		return nil, err
	}
	return []string{string(cert)}, nil
}

func (ca *DubboCA) GetCAKeyCertBundle() *util.KeyCertBundle {
	return ca.keyCertBundle
}

func (ca *DubboCA) sign(csrPEM []byte, subjectIDs []string, requestedLifetime time.Duration, checkLifetime, forCA bool) ([]byte, error) {
	signingCert, signingKey, _, _ := ca.keyCertBundle.GetAll()
	if signingCert == nil {
		return nil, caerror.NewError(caerror.CANotReady, fmt.Errorf("Dubbo CA is not ready")) // nolint
	}

	csr, err := util.ParsePemEncodedCSR(csrPEM)
	if err != nil {
		return nil, caerror.NewError(caerror.CSRError, err)
	}

	if err := csr.CheckSignature(); err != nil {
		return nil, caerror.NewError(caerror.CSRError, err)
	}

	lifetime := requestedLifetime
	// If the requested requestedLifetime is non-positive, apply the default TTL.
	if requestedLifetime.Seconds() <= 0 {
		lifetime = ca.defaultCertTTL
	}
	// If checkLifetime is set and the requested TTL is greater than maxCertTTL, return an error
	if checkLifetime && requestedLifetime.Seconds() > ca.maxCertTTL.Seconds() {
		return nil, caerror.NewError(caerror.TTLError, fmt.Errorf(
			"requested TTL %s is greater than the max allowed TTL %s", requestedLifetime, ca.maxCertTTL))
	}

	certBytes, err := util.GenCertFromCSR(csr, signingCert, csr.PublicKey, *signingKey, subjectIDs, lifetime, forCA)
	if err != nil {
		return nil, caerror.NewError(caerror.CertGenError, err)
	}

	block := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}
	cert := pem.EncodeToMemory(block)

	return cert, nil
}

func (ca *DubboCA) signWithCertChain(csrPEM []byte, subjectIDs []string, requestedLifetime time.Duration, lifetimeCheck, forCA bool) ([]byte, error) {
	cert, err := ca.sign(csrPEM, subjectIDs, requestedLifetime, lifetimeCheck, forCA)
	if err != nil {
		return nil, err
	}

	chainPem := ca.GetCAKeyCertBundle().GetCertChainPem()
	if len(chainPem) > 0 {
		cert = append(cert, chainPem...)
	}
	return cert, nil
}

func loadSelfSignedCaSecret(client corev1.CoreV1Interface, namespace string, caCertName string, rootCertFile string, caOpts *DubboCAOptions) error {
	caSecret, err := client.Secrets(namespace).Get(context.TODO(), caCertName, metav1.GetOptions{})
	if err == nil {
		pkiCaLog.Infof("Load signing key and cert from existing secret %s/%s", caSecret.Namespace, caSecret.Name)
		rootCerts, err := util.AppendRootCerts(caSecret.Data[CACertFile], rootCertFile)
		if err != nil {
			return fmt.Errorf("failed to append root certificates (%v)", err)
		}
		if caOpts.KeyCertBundle, err = util.NewVerifiedKeyCertBundleFromPem(
			caSecret.Data[CACertFile],
			caSecret.Data[CAPrivateKeyFile],
			nil,
			rootCerts,
			nil,
		); err != nil {
			return fmt.Errorf("failed to create CA KeyCertBundle (%v)", err)
		}
		pkiCaLog.Infof("Using existing public key: \n%v", string(rootCerts))
	}
	return err
}

func NewPluggedCertDubboCAOptions(fileBundle SigningCAFileBundle, defaultCertTTL, maxCertTTL time.Duration, caRSAKeySize int) (caOpts *DubboCAOptions, err error) {
	caOpts = &DubboCAOptions{
		CAType:         pluggedCertCA,
		DefaultCertTTL: defaultCertTTL,
		MaxCertTTL:     maxCertTTL,
		CARSAKeySize:   caRSAKeySize,
	}

	if caOpts.KeyCertBundle, err = util.NewVerifiedKeyCertBundleFromFile(
		fileBundle.SigningCertFile,
		fileBundle.SigningKeyFile,
		fileBundle.CertChainFiles,
		fileBundle.RootCertFile,
		fileBundle.CRLFile,
	); err != nil {
		return nil, fmt.Errorf("failed to create CA KeyCertBundle (%v)", err)
	}

	// Validate that the passed in signing cert can be used as CA.
	// The check can't be done inside `KeyCertBundle`, since bundle could also be used to
	// validate workload certificates (i.e., where the leaf certificate is not a CA).
	b, err := os.ReadFile(fileBundle.SigningCertFile)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM encoded certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse X.509 certificate")
	}
	if !cert.IsCA {
		return nil, fmt.Errorf("certificate is not authorized to sign other certificates")
	}

	return caOpts, nil
}

func (ca *DubboCA) GenKeyCert(hostnames []string, certTTL time.Duration, checkLifetime bool) ([]byte, []byte, error) {
	opts := util.CertOptions{
		RSAKeySize: rsaKeySize,
	}

	// use the type of private key the CA uses to generate an intermediate CA of that type (e.g. CA cert using RSA will
	// cause intermediate CAs using RSA to be generated)
	_, signingKey, _, _ := ca.keyCertBundle.GetAll()
	curve, err := util.GetEllipticCurve(signingKey)
	if err == nil {
		opts.ECSigAlg = util.EcdsaSigAlg
		switch curve {
		case elliptic.P384():
			opts.ECCCurve = util.P384Curve
		default:
			opts.ECCCurve = util.P256Curve
		}
	}

	csrPEM, privPEM, err := util.GenCSR(opts)
	if err != nil {
		return nil, nil, err
	}

	certPEM, err := ca.signWithCertChain(csrPEM, hostnames, certTTL, checkLifetime, false)
	if err != nil {
		return nil, nil, err
	}

	return certPEM, privPEM, nil
}

func (ca *DubboCA) minTTL(defaultCertTTL time.Duration) (time.Duration, error) {
	certChainPem := ca.keyCertBundle.GetCertChainPem()
	if len(certChainPem) == 0 {
		return defaultCertTTL, nil
	}

	certChainExpiration, err := util.TimeBeforeCertExpires(certChainPem, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to get cert chain TTL %s", err.Error())
	}

	if certChainExpiration <= 0 {
		return 0, fmt.Errorf("cert chain has expired")
	}

	if defaultCertTTL > certChainExpiration {
		return certChainExpiration, nil
	}

	return defaultCertTTL, nil
}

func NewSelfSignedDebugDubboCAOptions(rootCertFile string, caCertTTL, defaultCertTTL, maxCertTTL time.Duration, org string, caRSAKeySize int) (caOpts *DubboCAOptions, err error) {
	caOpts = &DubboCAOptions{
		CAType:         selfSignedCA,
		DefaultCertTTL: defaultCertTTL,
		MaxCertTTL:     maxCertTTL,
		CARSAKeySize:   caRSAKeySize,
	}

	options := util.CertOptions{
		TTL:          caCertTTL,
		Org:          org,
		IsCA:         true,
		IsSelfSigned: true,
		RSAKeySize:   caRSAKeySize,
		IsDualUse:    true, // hardcoded to true for K8S as well
	}
	pemCert, pemKey, ckErr := util.GenCertKeyFromOptions(options)
	if ckErr != nil {
		return nil, fmt.Errorf("unable to generate CA cert and key for self-signed CA (%v)", ckErr)
	}

	rootCerts, err := util.AppendRootCerts(pemCert, rootCertFile)
	if err != nil {
		return nil, fmt.Errorf("failed to append root certificates (%v)", err)
	}

	if caOpts.KeyCertBundle, err = util.NewVerifiedKeyCertBundleFromPem(
		pemCert,
		pemKey,
		nil,
		rootCerts,
		nil,
	); err != nil {
		return nil, fmt.Errorf("failed to create CA KeyCertBundle (%v)", err)
	}

	return caOpts, nil
}

func BuildSecret(scrtName, namespace string, certChain, privateKey, rootCert, caCert, caPrivateKey []byte, secretType v1.SecretType) *v1.Secret {
	secret := &v1.Secret{
		Data: map[string][]byte{
			CertChainFile:    certChain,
			PrivateKeyFile:   privateKey,
			RootCertFile:     rootCert,
			CACertFile:       caCert,
			CAPrivateKeyFile: caPrivateKey,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      scrtName,
			Namespace: namespace,
		},
		Type: secretType,
	}

	if scrtName == CACertsSecret {
		secret.Data[DubboGenerated] = []byte("")
	}

	return secret
}
