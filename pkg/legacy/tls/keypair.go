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
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
)

import (
	"github.com/pkg/errors"
)

type KeyPair struct {
	CertPEM []byte
	KeyPEM  []byte
}

func ToKeyPair(key crypto.PrivateKey, cert []byte) (*KeyPair, error) {
	keyPem, err := pemEncodeKey(key)
	if err != nil {
		return nil, errors.Wrap(err, "failed to PEM encode a private key")
	}
	certPem, err := pemEncodeCert(cert)
	if err != nil {
		return nil, errors.Wrap(err, "failed to PEM encode a certificate")
	}
	return &KeyPair{
		CertPEM: certPem,
		KeyPEM:  keyPem,
	}, nil
}

func pemEncodeKey(priv crypto.PrivateKey) ([]byte, error) {
	var block *pem.Block
	switch k := priv.(type) {
	case *ecdsa.PrivateKey:
		bytes, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			return nil, err
		}
		block = &pem.Block{Type: "EC PRIVATE KEY", Bytes: bytes}
	case *rsa.PrivateKey:
		block = &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	default:
		return nil, errors.Errorf("unsupported private key type %T", priv)
	}
	var keyBuf bytes.Buffer
	if err := pem.Encode(&keyBuf, block); err != nil {
		return nil, err
	}
	return keyBuf.Bytes(), nil
}

func pemEncodeCert(derBytes []byte) ([]byte, error) {
	var certBuf bytes.Buffer
	if err := pem.Encode(&certBuf, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return nil, err
	}
	return certBuf.Bytes(), nil
}
