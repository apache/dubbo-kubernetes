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

package features

import "github.com/apache/dubbo-kubernetes/pkg/env"

var (
	CertSignerDomain          = env.Register("CERT_SIGNER_DOMAIN", "", "The cert signer domain info").Get()
	UseCacertsForSelfSignedCA = env.Register("USE_CACERTS_FOR_SELF_SIGNED_CA", false,
		"If enabled, dubbod will use a secret named cacerts to store its self-signed dubbo-"+
			"generated root certificate.").Get()
	EnableXDSIdentityCheck = env.Register(
		"PLANET_ENABLE_XDS_IDENTITY_CHECK",
		true,
		"If enabled, planet will authorize XDS clients, to ensure they are acting only as namespaces they have permissions for.",
	).Get()
)
