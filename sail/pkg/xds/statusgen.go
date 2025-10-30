package xds

import (
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	v3 "github.com/apache/dubbo-kubernetes/sail/pkg/xds/v3"
)

const (
	TypeDebugPrefix = v3.DebugType + "/"
)

type StatusGen struct {
	Server *DiscoveryServer

	// TODO: track last N Nacks and connection events, with 'version' based on timestamp.
	// On new connect, use version to send recent events since last update.
}

func NewStatusGen(s *DiscoveryServer) *StatusGen {
	return &StatusGen{
		Server: s,
	}
}

func (sg *StatusGen) Generate(proxy *model.Proxy, w *model.WatchedResource, req *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	return sg.handleInternalRequest(proxy, w, req)
}

func (sg *StatusGen) handleInternalRequest(_ *model.Proxy, w *model.WatchedResource, _ *model.PushRequest) (model.Resources, model.XdsLogDetails, error) {
	res := model.Resources{}
	return res, model.DefaultXdsLogDetails, nil
}
