package gateway

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
)

type Gateway struct {
	*config.Config `json:"config"`
}

func (g Gateway) ResourceName() string {
	return config.NamespacedName(g.Config).String()
}
