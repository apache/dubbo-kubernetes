package xds

import (
	"github.com/apache/dubbo-kubernetes/navigator/pkg/model"
)

type DiscoveryServer struct {
}

func NewDiscoveryServer(env *model.Environment, clusterAliases map[string]string) *DiscoveryServer {
	out := &DiscoveryServer{}
	return out
}
