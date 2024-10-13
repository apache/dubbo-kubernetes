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

package v3

import (
	"bytes"
	"encoding/pem"
	"io"

	envoy_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
)

// NewServerCertificateSecret populates a new Envoy TLS certificate
// secret containing the given key and chain of certificates.
func NewServerCertificateSecret(key *pem.Block, certificates []*pem.Block) *envoy_auth.Secret {
	mustEncode := func(out io.Writer, b *pem.Block) {
		if err := pem.Encode(out, b); err != nil {
			panic(err.Error())
		}
	}

	keyBytes := &bytes.Buffer{}
	certificateBytes := &bytes.Buffer{}

	mustEncode(keyBytes, key)

	for _, c := range certificates {
		mustEncode(certificateBytes, c)
		certificateBytes.WriteString("\n")
	}

	return &envoy_auth.Secret{
		Type: &envoy_auth.Secret_TlsCertificate{
			TlsCertificate: &envoy_auth.TlsCertificate{
				CertificateChain: &envoy_core.DataSource{
					Specifier: &envoy_core.DataSource_InlineString{
						InlineString: certificateBytes.String(),
					},
				},
				PrivateKey: &envoy_core.DataSource{
					Specifier: &envoy_core.DataSource_InlineString{
						InlineString: keyBytes.String(),
					},
				},
			},
		},
	}
}
