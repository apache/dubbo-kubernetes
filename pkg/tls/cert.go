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

package tls

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	util_rsa "github.com/apache/dubbo-kubernetes/pkg/util/rsa"
	"github.com/pkg/errors"
	"math/big"
	"net"
	"time"
)

var DefaultValidityPeriod = 10 * 365 * 24 * time.Hour

type CertType string

const (
	ServerCertType              CertType = "server"
	ClientCertType              CertType = "client"
	DefaultAllowedClockSkew              = 5 * time.Minute
	DefaultCACertValidityPeriod          = 10 * 365 * 24 * time.Hour
)

type KeyType func() (crypto.Signer, error)

var ECDSAKeyType KeyType = func() (crypto.Signer, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

var RSAKeyType KeyType = func() (crypto.Signer, error) {
	return util_rsa.GenerateKey(util_rsa.DefaultKeySize)
}

var DefaultKeyType = RSAKeyType

func NewSelfSignedCert(certType CertType, keyType KeyType, hosts ...string) (KeyPair, error) {
	key, err := keyType()
	if err != nil {
		return KeyPair{}, errors.Wrap(err, "failed to generate TLS key")
	}

	csr, err := newCert(nil, certType, hosts...)
	if err != nil {
		return KeyPair{}, err
	}
	certDerBytes, err := x509.CreateCertificate(rand.Reader, &csr, &csr, key.Public(), key)
	if err != nil {
		return KeyPair{}, errors.Wrap(err, "failed to generate TLS certificate")
	}

	certBytes, err := pemEncodeCert(certDerBytes)
	if err != nil {
		return KeyPair{}, err
	}

	keyBytes, err := pemEncodeKey(key)
	if err != nil {
		return KeyPair{}, err
	}

	return KeyPair{
		CertPEM: certBytes,
		KeyPEM:  keyBytes,
	}, nil
}

// NewCert generates certificate that is signed by the CA (parent)
func NewCert(
	parent x509.Certificate,
	parentKey crypto.Signer,
	certType CertType,
	keyType KeyType,
	hosts ...string,
) (KeyPair, error) {
	key, err := keyType()
	if err != nil {
		return KeyPair{}, errors.Wrap(err, "failed to generate TLS key")
	}

	csr, err := newCert(&parent.Subject, certType, hosts...)
	if err != nil {
		return KeyPair{}, err
	}

	certDerBytes, err := x509.CreateCertificate(rand.Reader, &csr, &parent, key.Public(), parentKey)
	if err != nil {
		return KeyPair{}, errors.Wrap(err, "failed to generate TLS certificate")
	}

	certBytes, err := pemEncodeCert(certDerBytes)
	if err != nil {
		return KeyPair{}, err
	}

	keyBytes, err := pemEncodeKey(key)
	if err != nil {
		return KeyPair{}, err
	}

	return KeyPair{
		CertPEM: certBytes,
		KeyPEM:  keyBytes,
	}, nil
}

func newCert(issuer *pkix.Name, certType CertType, hosts ...string) (x509.Certificate, error) {
	notBefore := time.Now()
	notAfter := notBefore.Add(DefaultValidityPeriod)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return x509.Certificate{}, errors.Wrap(err, "failed to generate serial number")
	}
	csr := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  issuer == nil,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{},
		BasicConstraintsValid: true,
	}
	if issuer != nil {
		csr.Issuer = *issuer
	} else {
		// root ca
		csr.KeyUsage |= x509.KeyUsageCertSign
	}
	switch certType {
	case ServerCertType:
		csr.ExtKeyUsage = append(csr.ExtKeyUsage, x509.ExtKeyUsageServerAuth)
	case ClientCertType:
		csr.ExtKeyUsage = append(csr.ExtKeyUsage, x509.ExtKeyUsageClientAuth)
	default:
		return x509.Certificate{}, errors.Errorf("invalid certificate type %q, expected either %q or %q",
			certType, ServerCertType, ClientCertType)
	}
	for _, host := range hosts {
		if ip := net.ParseIP(host); ip != nil {
			csr.IPAddresses = append(csr.IPAddresses, ip)
		} else {
			csr.DNSNames = append(csr.DNSNames, host)
		}
	}
	return csr, nil
}

func GenerateCA(keyType KeyType, subject pkix.Name) (*KeyPair, error) {
	key, err := keyType()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate a private key")
	}

	now := core.Now()
	notBefore := now.Add(-DefaultAllowedClockSkew)
	notAfter := now.Add(DefaultCACertValidityPeriod)
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(0),
		Subject:               subject,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		PublicKey:             key.Public(),
	}

	ca, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, key.Public(), key)
	if err != nil {
		return nil, err
	}

	return ToKeyPair(key, ca)
}
