package credentialfetcher

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/security"
	"github.com/apache/dubbo-kubernetes/security/pkg/credentialfetcher/plugin"
)

func NewCredFetcher(credtype string) (security.CredFetcher, error) {
	switch credtype {
	case security.JWT, "":
		return plugin.CreateTokenPlugin(), nil
	default:
		return nil, fmt.Errorf("invalid credential fetcher type %s", credtype)
	}
}
