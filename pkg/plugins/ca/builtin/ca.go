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

package builtin

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net/url"
	"time"

	core_ca "github.com/apache/dubbo-kubernetes/pkg/core/ca"
	util_tls "github.com/apache/dubbo-kubernetes/pkg/tls"
	util_rsa "github.com/apache/dubbo-kubernetes/pkg/util/rsa"
	"github.com/pkg/errors"

	"github.com/spiffe/go-spiffe/v2/spiffeid"

	"github.com/apache/dubbo-kubernetes/pkg/core"
)

const (
	DefaultAllowedClockSkew     = 10 * time.Second
	DefaultCACertValidityPeriod = 10 * 365 * 24 * time.Hour
)

type certOptsFn = func(*x509.Certificate)

func withExpirationTime(expiration time.Duration) certOptsFn {
	return func(certificate *x509.Certificate) {
		now := core.Now()
		certificate.NotAfter = now.Add(expiration)
	}
}

func newRootCa(mesh string, rsaBits int, certOpts ...certOptsFn) (*core_ca.KeyPair, error) {
	if rsaBits == 0 {
		rsaBits = util_rsa.DefaultKeySize
	}
	key, err := util_rsa.GenerateKey(rsaBits)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate a private key")
	}
	cert, err := newCACert(key, mesh, certOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate X509 certificate")
	}
	return util_tls.ToKeyPair(key, cert)
}

func newCACert(signer crypto.Signer, trustDomain string, certOpts ...certOptsFn) ([]byte, error) {
	subject := pkix.Name{
		Organization:       []string{"Dubbo"},
		OrganizationalUnit: []string{"Mesh"},
		CommonName:         trustDomain,
	}
	now := core.Now()
	notBefore := now.Add(-DefaultAllowedClockSkew)
	notAfter := now.Add(DefaultCACertValidityPeriod)

	template, err := caTemplate(trustDomain, subject, signer.Public(), notBefore, notAfter, big.NewInt(0))
	if err != nil {
		return nil, err
	}

	for _, opt := range certOpts {
		opt(template)
	}
	return x509.CreateCertificate(rand.Reader, template, template, signer.Public(), signer)
}

func caTemplate(trustDomain string, subject pkix.Name, publicKey crypto.PublicKey, notBefore, notAfter time.Time, serialNumber *big.Int) (*x509.Certificate, error) {
	domain, err := spiffeid.TrustDomainFromString(trustDomain)
	if err != nil {
		return nil, err
	}
	uri, err := spiffeid.FromSegments(domain)
	if err != nil {
		return nil, err
	}
	return &x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      subject,
		URIs:         []*url.URL{uri.URL()},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage: x509.KeyUsageCertSign |
			x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		PublicKey:             publicKey,
	}, nil
}
