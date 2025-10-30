package options

import (
	dubboagent "github.com/apache/dubbo-kubernetes/pkg/dubbo-agent"
	meshconfig "istio.io/api/mesh/v1alpha1"
	"os"
	"path/filepath"
	"strings"
)

const xdsHeaderPrefix = "XDS_HEADER_"

func NewAgentOptions(proxy *ProxyArgs, cfg *meshconfig.ProxyConfig, sds dubboagent.SDSServiceFactory) *dubboagent.AgentOptions {
	o := &dubboagent.AgentOptions{
		XDSRootCerts:               xdsRootCA,
		CARootCerts:                caRootCA,
		XDSHeaders:                 map[string]string{},
		XdsUdsPath:                 filepath.Join(cfg.ConfigPath, "XDS"),
		GRPCBootstrapPath:          grpcBootstrapEnv,
		ProxyType:                  proxy.Type,
		ServiceNode:                proxy.ServiceNode(),
		EnableDynamicProxyConfig:   enableProxyConfigXdsEnv,
		SDSFactory:                 sds,
		WorkloadIdentitySocketFile: workloadIdentitySocketFile,
		ProxyIPAddresses:           proxy.IPAddresses,
		ProxyDomain:                proxy.DNSDomain,
		DubbodSAN:                  dubbodSAN.Get(),
	}
	extractXDSHeadersFromEnv(o)
	return o
}

func extractXDSHeadersFromEnv(o *dubboagent.AgentOptions) {
	envs := os.Environ()
	for _, e := range envs {
		if strings.HasPrefix(e, xdsHeaderPrefix) {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) != 2 {
				continue
			}
			o.XDSHeaders[parts[0][len(xdsHeaderPrefix):]] = parts[1]
		}
	}
}
