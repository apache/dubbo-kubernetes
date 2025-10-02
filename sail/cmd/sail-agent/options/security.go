package options

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/jwt"
	"github.com/apache/dubbo-kubernetes/pkg/security"
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

	o, err := SetupSecurityOptions(proxyConfig, o, jwtPolicy.Get())
	if err != nil {
		return o, err
	}

	extractCAHeadersFromEnv(o)

	return o, err
}

func SetupSecurityOptions(proxyConfig *meshconfig.ProxyConfig, secOpt *security.Options, jwtPolicy string) (*security.Options, error) {
	switch jwtPolicy {
	case jwt.PolicyFirstParty:
		klog.Warningf("Using deprecated JWT policy 'first-party-jwt'; treating as 'third-party-jwt'")
	default:
		klog.Info("Using existing certs")
	}
	o := secOpt

	// If not set explicitly, default to the discovery address.
	if o.CAEndpoint == "" {
		o.CAEndpoint = proxyConfig.DiscoveryAddress
		o.CAEndpointSAN = dubbodSAN.Get()
	}

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
