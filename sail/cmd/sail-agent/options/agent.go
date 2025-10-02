package options

import (
	dubboagent "github.com/apache/dubbo-kubernetes/pkg/dubbo-agent"
	meshconfig "istio.io/api/mesh/v1alpha1"
	"os"
	"strings"
)

const xdsHeaderPrefix = "XDS_HEADER_"

func NewAgentOptions(proxy *ProxyArgs, cfg *meshconfig.ProxyConfig, sds dubboagent.SDSServiceFactory) *dubboagent.AgentOptions {
	o := &dubboagent.AgentOptions{
		XDSHeaders:                 map[string]string{},
		GRPCBootstrapPath:          grpcBootstrapEnv,
		SDSFactory:                 sds,
		WorkloadIdentitySocketFile: workloadIdentitySocketFile,
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
