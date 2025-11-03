package features

import "github.com/apache/dubbo-kubernetes/pkg/env"

var (
	CACertConfigMapName = env.Register("SAIL_CA_CERT_CONFIGMAP", "dubbo-ca-root-cert",
		"The name of the ConfigMap that stores the Root CA Certificate that is used by dubbod").Get()
	EnableLeaderElection = env.Register("ENABLE_LEADER_ELECTION", true,
		"If enabled (default), starts a leader election client and gains leadership before executing controllers. "+
			"If false, it assumes that only one instance of istiod is running and skips leader election.").Get()
)
