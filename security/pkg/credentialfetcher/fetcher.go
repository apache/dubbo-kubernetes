package credentialfetcher

import (
	"fmt"

	"github.com/apache/dubbo-kubernetes/pkg/security"
	"github.com/apache/dubbo-kubernetes/security/pkg/credentialfetcher/plugin"
)

func NewCredFetcher(credtype, trustdomain, jwtPath, identityProvider string) (security.CredFetcher, error) {
	switch credtype {
	case security.JWT, "":
		// If unset, also default to JWT for backwards compatibility
		if jwtPath == "" {
			return nil, nil // no cred fetcher - using certificates only
		}
		return plugin.CreateTokenPlugin(jwtPath), nil
	default:
		return nil, fmt.Errorf("invalid credential fetcher type %s", credtype)
	}
}
