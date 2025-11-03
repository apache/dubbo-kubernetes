package xds

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/apache/dubbo-kubernetes/pkg/env"
	"github.com/apache/dubbo-kubernetes/pkg/lazy"
	dubboversion "github.com/apache/dubbo-kubernetes/pkg/version"
	"github.com/apache/dubbo-kubernetes/pkg/xds"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"github.com/apache/dubbo-kubernetes/sail/pkg/networking/util"
	v3 "github.com/apache/dubbo-kubernetes/sail/pkg/xds/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"k8s.io/klog/v2"
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
	if typ != TypeDebugSyncronization {
		// Currently only TypeDebugSyncronization utilizes this so don't both sending otherwise
		return nil
	}
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
		ControlPlane: ControlPlane(w.TypeUrl),
		TypeUrl:      w.TypeUrl,
		VersionInfo:  req.Push.PushVersion,
		Nonce:        nonce(req.Push.PushVersion),
		Resources:    xds.ResourcesToAny(res),
	}

	ptype := "PUSH"
	if logdata.Incremental {
		ptype = "PUSH INC"
	}

	if err := xds.Send(con, resp); err != nil {
		return err
	}

	switch {
	case !req.Full:
	default:
		// Log format matches Istio: "LDS: PUSH for node:xxx resources:1 size:342B"
		klog.Infof("%s: %s for node:%s resources:%d size:%s", v3.GetShortType(w.TypeUrl), ptype, con.proxy.ID, len(res),
			util.ByteCount(ResourceSize(res)))
	}

	return nil
}

func ResourceSize(r model.Resources) int {
	size := 0
	for _, r := range r {
		size += len(r.Resource.Value)
	}
	return size
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

// atMostNJoin joins up to N items from data, appending "and X others" if more exist
func atMostNJoin(data []string, limit int) string {
	if limit == 0 || limit == 1 {
		// Assume limit >1, but make sure we don't crash if someone does pass those
		return strings.Join(data, ", ")
	}
	if len(data) == 0 {
		return ""
	}
	if len(data) < limit {
		return strings.Join(data, ", ")
	}
	return strings.Join(data[:limit-1], ", ") + fmt.Sprintf(", and %d others", len(data)-limit+1)
}
