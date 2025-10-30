package xds

import (
	"encoding/json"
	"github.com/apache/dubbo-kubernetes/pkg/env"
	"github.com/apache/dubbo-kubernetes/pkg/lazy"
	dubboversion "github.com/apache/dubbo-kubernetes/pkg/version"
	"github.com/apache/dubbo-kubernetes/pkg/xds"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"k8s.io/klog/v2"
	"strconv"
	"strings"
)

type DubboControlPlaneInstance struct {
	Component string
	ID        string
	Info      dubboversion.BuildInfo
}

var controlPlane = lazy.New(func() (*core.ControlPlane, error) {
	podName := env.Register("POD_NAME", "", "").Get()
	byVersion, err := json.Marshal(DubboControlPlaneInstance{
		Component: "dubbod",
		ID:        podName,
		Info:      dubboversion.Info,
	})
	if err != nil {
		klog.Warningf("XDS: Could not serialize control plane id: %v", err)
	}
	return &core.ControlPlane{Identifier: string(byVersion)}, nil
})

func ControlPlane(typ string) *core.ControlPlane {
	cp, _ := controlPlane.Get()
	return cp
}

func xdsNeedsPush(req *model.PushRequest, _ *model.Proxy) (needsPush, definitive bool) {
	if req == nil {
		return true, true
	}
	if req.Forced {
		return true, true
	}
	return false, false
}
func (s *DiscoveryServer) pushXds(con *Connection, w *model.WatchedResource, req *model.PushRequest) error {
	if w == nil {
		return nil
	}
	gen := s.findGenerator(w.TypeUrl, con)
	if gen == nil {
		return nil
	}

	// If delta is set, client is requesting new resources or removing old ones. We should just generate the
	// new resources it needs, rather than the entire set of known resources.
	// Note: we do not need to account for unsubscribed resources as these are handled by parent removal;
	// See https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol#deleting-resources.
	// This means if there are only removals, we will not respond.
	var logFiltered string
	if !req.Delta.IsEmpty() && !con.proxy.IsProxylessGrpc() {
		logFiltered = " filtered:" + strconv.Itoa(len(w.ResourceNames)-len(req.Delta.Subscribed))
		w = &model.WatchedResource{
			TypeUrl:       w.TypeUrl,
			ResourceNames: req.Delta.Subscribed,
		}
	}

	res, logdata, err := gen.Generate(con.proxy, w, req)
	info := ""
	if len(logdata.AdditionalInfo) > 0 {
		info = " " + logdata.AdditionalInfo
	}
	if len(logFiltered) > 0 {
		info += logFiltered
	}
	if err != nil || res == nil {
		return err
	}

	resp := &discovery.DiscoveryResponse{
		TypeUrl:   w.TypeUrl,
		Resources: xds.ResourcesToAny(res),
	}

	if err := xds.Send(con, resp); err != nil {
		return err
	}

	return nil
}

func (s *DiscoveryServer) findGenerator(typeURL string, con *Connection) model.XdsResourceGenerator {
	if g, f := s.Generators[con.proxy.Metadata.Generator+"/"+typeURL]; f {
		return g
	}

	if g, f := s.Generators[typeURL]; f {
		return g
	}

	// XdsResourceGenerator is the default generator for this connection. We want to allow
	// some types to use custom generators - for example EDS.
	g := con.proxy.XdsResourceGenerator
	if g == nil {
		if strings.HasPrefix(typeURL, TypeDebugPrefix) {
			g = s.Generators["event"]
		} else {
			// TODO move this to just directly using the resource TypeUrl
			g = s.Generators["api"] // default to "MCP" generators - any type supported by store
		}
	}
	return g
}
