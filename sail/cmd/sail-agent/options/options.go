package options

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/env"
	"github.com/apache/dubbo-kubernetes/pkg/jwt"
	"github.com/apache/dubbo-kubernetes/pkg/security"
	"path/filepath"
)

var (
	dubbodSAN = env.Register("DUBBOD_SAN", "",
		"Override the ServerName used to validate Dubbod certificate. "+
			"Can be used as an alternative to setting /etc/hosts for VMs - discovery address will be an IP:port")
	jwtPolicy = env.Register("JWT_POLICY", jwt.PolicyThirdParty,
		"The JWT validation policy.")
	workloadIdentitySocketFile = env.Register("WORKLOAD_IDENTITY_SOCKET_FILE", security.DefaultWorkloadIdentitySocketFile,
		fmt.Sprintf("SPIRE workload identity SDS socket filename. If set, an SDS socket with this name must exist at %s", security.WorkloadIdentityPath)).Get()
	grpcBootstrapEnv = env.Register("GRPC_XDS_BOOTSTRAP", filepath.Join(constants.ConfigPathDir, "grpc-bootstrap.json"),
		"Path where gRPC expects to read a bootstrap file. Agent will generate one if set.").Get()
	caProviderEnv = env.Register("CA_PROVIDER", "Aegis", "name of authentication provider").Get()
	caEndpointEnv = env.Register("CA_ADDR", "", "Address of the spiffe certificate provider. Defaults to discoveryAddress").Get()
)
