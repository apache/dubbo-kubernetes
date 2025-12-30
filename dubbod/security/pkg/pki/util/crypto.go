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

package util

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"reflect"
	"strings"
)

const (
	blockTypeECPrivateKey    = "EC PRIVATE KEY"
	blockTypeRSAPrivateKey   = "RSA PRIVATE KEY" // PKCS#1 private key
	blockTypePKCS8PrivateKey = "PRIVATE KEY"     // PKCS#8 plain private key
)

func ParsePemEncodedCertificate(certBytes []byte) (*x509.Certificate, error) {
	cb, _ := pem.Decode(certBytes)
	if cb == nil {
		return nil, fmt.Errorf("invalid PEM encoded certificate")
	}

	cert, err := x509.ParseCertificate(cb.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse X.509 certificate")
	}

	return cert, nil
}

func ParsePemEncodedCertificateChain(certBytes []byte) ([]*x509.Certificate, []byte, error) {
	var (
		certs         []*x509.Certificate
		cb            *pem.Block
		rootCertBytes []byte
	)
	certBytes = bytes.TrimSpace(certBytes)
	for {
		rootCertBytes = certBytes
		cb, certBytes = pem.Decode(certBytes)
		if cb == nil {
			return nil, nil, fmt.Errorf("invalid PEM encoded certificate")
		}
		cert, err := x509.ParseCertificate(cb.Bytes)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse X.509 certificate : %v", err)
		}
		certs = append(certs, cert)
		if len(certBytes) == 0 {
			break
		}
	}
	if len(certs) == 0 {
		return nil, nil, fmt.Errorf("no PEM encoded X.509 certificates parsed")
	}
	return certs, rootCertBytes, nil
}

func ParsePemEncodedCSR(csrBytes []byte) (*x509.CertificateRequest, error) {
	block, _ := pem.Decode(csrBytes)
	if block == nil {
		return nil, fmt.Errorf("certificate signing request is not properly encoded")
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse X.509 certificate signing request")
	}
	return csr, nil
}

func ParsePemEncodedKey(keyBytes []byte) (crypto.PrivateKey, error) {
	kb, _ := pem.Decode(keyBytes)
	if kb == nil {
		return nil, fmt.Errorf("invalid PEM-encoded key")
	}

	switch kb.Type {
	case blockTypeECPrivateKey:
		key, err := x509.ParseECPrivateKey(kb.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse the ECDSA private key: %v", err)
		}
		return key, nil
	case blockTypeRSAPrivateKey:
		key, err := x509.ParsePKCS1PrivateKey(kb.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse the RSA private key: %v", err)
		}
		return key, nil
	case blockTypePKCS8PrivateKey:
		key, err := x509.ParsePKCS8PrivateKey(kb.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse the PKCS8 private key: %v", err)
		}
		return key, nil
	default:
		return nil, fmt.Errorf("unsupported PEM block type for a private key: %s", kb.Type)
	}
}

func GetRSAKeySize(privKey crypto.PrivateKey) (int, error) {
	if t := reflect.TypeOf(privKey); t != reflect.TypeOf(&rsa.PrivateKey{}) {
		return 0, fmt.Errorf("key type is not RSA: %v", t)
	}
	pkey := privKey.(*rsa.PrivateKey)
	return pkey.N.BitLen(), nil
}

func GetEllipticCurve(privKey *crypto.PrivateKey) (elliptic.Curve, error) {
	switch key := (*privKey).(type) {
	// this should agree with var SupportedECSignatureAlgorithms
	case *ecdsa.PrivateKey:
		if key.Curve == elliptic.P384() {
			return key.Curve, nil
		}
		return elliptic.P256(), nil
	default:
		return nil, fmt.Errorf("private key is not ECDSA based")
	}
}

func PemCertBytestoString(caCerts []byte) []string {
	certs := []string{}
	var cert string
	pemBlock := caCerts
	for block, rest := pem.Decode(pemBlock); block != nil && len(block.Bytes) != 0; block, rest = pem.Decode(pemBlock) {
		if len(rest) == 0 {
			cert = strings.TrimPrefix(strings.TrimSuffix(string(pemBlock), "\n"), "\n")
			certs = append(certs, cert)
			break
		}
		cert = string(pemBlock[0 : len(pemBlock)-len(rest)])
		cert = strings.TrimPrefix(strings.TrimSuffix(cert, "\n"), "\n")
		certs = append(certs, cert)
		pemBlock = rest
	}
	return certs
}
