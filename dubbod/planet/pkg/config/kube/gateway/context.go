package gateway

import (
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
)

type Context struct {
	ps      *model.PushContext
	cluster cluster.ID
}

func NewGatewayContext(ps *model.PushContext, cluster cluster.ID) Context {
	return Context{ps, cluster}
}
