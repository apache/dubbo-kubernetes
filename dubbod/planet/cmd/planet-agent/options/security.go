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

package options

import (
	"fmt"
	"os"
	"strings"

	serviceRouteIndex "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/features"
	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/credentialfetcher"
	"github.com/apache/dubbo-kubernetes/dubbod/security/pkg/nodeagent/cafile"
	"github.com/apache/dubbo-kubernetes/pkg/jwt"
	"github.com/apache/dubbo-kubernetes/pkg/security"

	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"
)

var log = dubbolog.RegisterScope("security", "security options debugging")

const caHeaderPrefix = "CA_HEADER_"

func NewSecurityOptions(proxyConfig *serviceRouteIndex.ProxyConfig, stsPort int, tokenManagerPlugin string) (*security.Options, error) {
	o := &security.Options{
		CAEndpoint:           caEndpointEnv,
		CAProviderName:       caProviderEnv,
		PlanetCertProvider:   features.PlanetCertProvider,
		OutputKeyCertToDir:   outputKeyCertToDir,
		ProvCert:             provCert,
		ClusterID:            clusterIDVar.Get(),
		FileMountedCerts:     fileMountedCertsEnv,
		WorkloadNamespace:    PodNamespaceVar.Get(),
		ServiceAccount:       serviceAccountVar.Get(),
		XdsAuthProvider:      xdsAuthProvider.Get(),
		TrustDomain:          trustDomainEnv,
		WorkloadRSAKeySize:   workloadRSAKeySizeEnv,
		Pkcs8Keys:            pkcs8KeysEnv,
		ECCSigAlg:            eccSigAlgEnv,
		ECCCurve:             eccCurvEnv,
		SecretTTL:            secretTTLEnv,
		FileDebounceDuration: fileDebounceDuration,
		STSPort:              stsPort,
		CertSigner:           certSigner.Get(),
		CARootPath:           cafile.CACertFilePath,
		CertChainFilePath:    security.DefaultCertChainFilePath,
		KeyFilePath:          security.DefaultKeyFilePath,
		RootCertFilePath:     security.DefaultRootCertFilePath,
		CAHeaders:            map[string]string{},
	}

	o, err := SetupSecurityOptions(proxyConfig, o, jwtPolicy.Get(),
		credFetcherTypeEnv, credIdentityProvider)
	if err != nil {
		return o, err
	}

	extractCAHeadersFromEnv(o)

	return o, nil
}

func SetupSecurityOptions(proxyConfig *serviceRouteIndex.ProxyConfig, secOpt *security.Options, jwtPolicy,
	credFetcherTypeEnv, credIdentityProvider string,
) (*security.Options, error) {
	// TODO jwtPath
	switch jwtPolicy {
	case jwt.PolicyThirdParty:
		log.Info("JWT policy is third-party-jwt")

	case jwt.PolicyFirstParty:
		log.Warnf("Using deprecated JWT policy 'first-party-jwt'; treating as 'third-party-jwt'")

	default:
		log.Info("Using existing certs")
	}

	o := secOpt

	// If not set explicitly, default to the discovery address.
	if o.CAEndpoint == "" {
		o.CAEndpoint = proxyConfig.DiscoveryAddress
		o.CAEndpointSAN = dubbodSAN.Get()
	}

	o.CredIdentityProvider = credIdentityProvider
	credFetcher, err := credentialfetcher.NewCredFetcher(credFetcherTypeEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to create credential fetcher: %v", err)
	}
	log.Infof("using credential fetcher of %s type in %s trust domain", credFetcherTypeEnv, o.TrustDomain)
	o.CredFetcher = credFetcher

	if o.ProvCert != "" && o.FileMountedCerts {
		return nil, fmt.Errorf("invalid options: PROV_CERT and FILE_MOUNTED_CERTS are mutually exclusive")
	}
	return o, nil
}

func extractCAHeadersFromEnv(o *security.Options) {
	envs := os.Environ()
	for _, e := range envs {
		if !strings.HasPrefix(e, caHeaderPrefix) {
			continue
		}

		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			continue
		}
		o.CAHeaders[parts[0][len(caHeaderPrefix):]] = parts[1]
	}
}
