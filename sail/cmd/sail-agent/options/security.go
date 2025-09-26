package options

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/jwt"
	"github.com/apache/dubbo-kubernetes/pkg/security"
	"github.com/apache/dubbo-kubernetes/security/pkg/credentialfetcher"
	meshconfig "istio.io/api/mesh/v1alpha1"
	"k8s.io/klog/v2"
	"os"
	"strings"
)

const caHeaderPrefix = "CA_HEADER_"

func NewSecurityOptions(proxyConfig *meshconfig.ProxyConfig, stsPort int, tokenManagerPlugin string) (*security.Options, error) {
	o := &security.Options{
		CAEndpoint:     caEndpointEnv,
		CAProviderName: caProviderEnv,
	}

	o, err := SetupSecurityOptions(proxyConfig, o, jwtPolicy.Get(),
		credFetcherTypeEnv, credIdentityProvider)
	if err != nil {
		return o, err
	}

	extractCAHeadersFromEnv(o)

	return o, err
}

func SetupSecurityOptions(proxyConfig *meshconfig.ProxyConfig, secOpt *security.Options, jwtPolicy,
	credFetcherTypeEnv, credIdentityProvider string,
) (*security.Options, error) {
	jwtPath := constants.ThirdPartyJwtPath
	switch jwtPolicy {
	case jwt.PolicyThirdParty:
		klog.Info("JWT policy is third-party-jwt")
		jwtPath = constants.ThirdPartyJwtPath
	case jwt.PolicyFirstParty:
		klog.Warningf("Using deprecated JWT policy 'first-party-jwt'; treating as 'third-party-jwt'")
		jwtPath = constants.ThirdPartyJwtPath
	default:
		klog.Info("Using existing certs")
	}

	o := secOpt

	// If not set explicitly, default to the discovery address.
	if o.CAEndpoint == "" {
		o.CAEndpoint = proxyConfig.DiscoveryAddress
		o.CAEndpointSAN = dubbodSAN.Get()
	}

	o.CredIdentityProvider = credIdentityProvider
	credFetcher, err := credentialfetcher.NewCredFetcher(credFetcherTypeEnv, o.TrustDomain, jwtPath, o.CredIdentityProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to create credential fetcher: %v", err)
	}
	klog.Infof("using credential fetcher of %s type in %s trust domain", credFetcherTypeEnv, o.TrustDomain)
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
