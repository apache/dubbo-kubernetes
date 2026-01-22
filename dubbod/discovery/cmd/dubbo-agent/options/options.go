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
	"path/filepath"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/jwt"

	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/env"
	"github.com/apache/dubbo-kubernetes/pkg/security"
)

var (
	InstanceIPVar   = env.Register("INSTANCE_IP", "", "")
	PodNameVar      = env.Register("POD_NAME", "", "")
	PodNamespaceVar = env.Register("POD_NAMESPACE", "", "")

	serviceAccountVar = env.Register("SERVICE_ACCOUNT", "", "Name of service account")

	xdsAuthProvider = env.Register("XDS_AUTH_PROVIDER", "", "Provider for XDS auth")

	caRootCA = env.Register("CA_ROOT_CA", "",
		"Explicitly set the root CA to expect for the CA connection.").Get()

	pkcs8KeysEnv = env.Register("PKCS8_KEY", false,
		"Whether to generate PKCS#8 private keys").Get()
	eccSigAlgEnv          = env.Register("ECC_SIGNATURE_ALGORITHM", "", "The type of ECC signature algorithm to use when generating private keys").Get()
	eccCurvEnv            = env.Register("ECC_CURVE", "P256", "The elliptic curve to use when ECC_SIGNATURE_ALGORITHM is set to ECDSA").Get()
	workloadRSAKeySizeEnv = env.Register("WORKLOAD_RSA_KEY_SIZE", 2048,
		"Specify the RSA key size to use for workload certificates.").Get()
	workloadIdentitySocketFile = env.Register("WORKLOAD_IDENTITY_SOCKET_FILE", security.DefaultWorkloadIdentitySocketFile,
		fmt.Sprintf("SPIRE workload identity SDS socket filename. If set, an SDS socket with this name must exist at %s", security.WorkloadIdentityPath)).Get()

	secretTTLEnv = env.Register("SECRET_TTL", 24*time.Hour,
		"The cert lifetime requested by dubbo agent").Get()

	fileDebounceDuration = env.Register("FILE_DEBOUNCE_DURATION", 100*time.Millisecond,
		"The duration for which the file read operation is delayed once file update is detected").Get()

	grpcBootstrapEnv = env.Register("GRPC_XDS_BOOTSTRAP", filepath.Join(constants.ConfigPathDir, "grpc-bootstrap.json"),
		"Path where gRPC expects to read a bootstrap file. Agent will generate one if set.").Get()

	caProviderEnv = env.Register("CA_PROVIDER", "Aegis", "name of authentication provider").Get()
	caEndpointEnv = env.Register("CA_ADDR", "", "Address of the spiffe certificate provider. Defaults to discoveryAddress").Get()

	clusterIDVar        = env.Register("DUBBO_META_CLUSTER_ID", "", "")
	fileMountedCertsEnv = env.Register("FILE_MOUNTED_CERTS", false, "").Get()

	trustDomainEnv = env.Register("TRUST_DOMAIN", "cluster.local",
		"The trust domain for spiffe certificates").Get()

	certSigner = env.Register("DUBBO_META_CERT_SIGNER", "",
		"The cert signer info for workload cert")

	ProxyConfigEnv = env.Register(
		"PROXY_CONFIG",
		"",
		"The proxy configuration. This will be set by the injection - gateways will use file mounts.",
	).Get()

	outputKeyCertToDir = env.Register("OUTPUT_CERTS", "",
		"The output directory for the key and certificate. If empty, key and certificate will not be saved. "+
			"Must be set for VMs using provisioning certificates.").Get()

	provCert = env.Register("PROV_CERT", "",
		"Set to a directory containing provisioned certs, for VMs").Get()

	dubbodSAN = env.Register("DUBBOD_SAN", "",
		"Override the ServerName used to validate Dubbod certificate. "+
			"Can be used as an alternative to setting /etc/hosts for VMs - discovery address will be an IP:port")

	jwtPolicy = env.Register("JWT_POLICY", jwt.PolicyThirdParty,
		"The JWT validation policy.")

	credFetcherTypeEnv = env.Register("CREDENTIAL_FETCHER_TYPE", security.JWT,
		"The type of the credential fetcher. Currently supported types include GoogleComputeEngine").Get()
	credIdentityProvider = env.Register("CREDENTIAL_IDENTITY_PROVIDER", "GoogleComputeEngine",
		"The identity provider for credential. Currently default supported identity provider is GoogleComputeEngine").Get()

	xdsRootCA = env.Register("XDS_ROOT_CA", "",
		"Explicitly set the root CA to expect for the XDS connection.").Get()

	enableProxyConfigXdsEnv = env.Register("PROXY_CONFIG_XDS_AGENT", false,
		"If set to true, agent retrieves dynamic proxy-config updates via xds channel").Get()
)
