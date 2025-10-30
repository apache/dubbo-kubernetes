package grpcgen

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/dubbo-agent/grpcxds"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"github.com/apache/dubbo-kubernetes/sail/pkg/networking/util"
	"github.com/apache/dubbo-kubernetes/sail/pkg/util/protoconv"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"k8s.io/klog/v2"
	"net"
	"strconv"
	"strings"
)

type listenerNames map[string]listenerName

type listenerName struct {
	RequestedNames sets.String
	Ports          sets.String
}

func newListenerNameFilter(names []string, node *model.Proxy) listenerNames {
	filter := make(listenerNames, len(names))
	for _, name := range names {
		// inbound, create a simple entry and move on
		if strings.HasPrefix(name, grpcxds.ServerListenerNamePrefix) {
			filter[name] = listenerName{RequestedNames: sets.New(name)}
			continue
		}

		host, port, err := net.SplitHostPort(name)
		hasPort := err == nil

		// attempt to expand shortname to FQDN
		requestedName := name
		if hasPort {
			requestedName = host
		}
		allNames := []string{requestedName}
		if fqdn := tryFindFQDN(requestedName, node); fqdn != "" {
			allNames = append(allNames, fqdn)
		}

		for _, name := range allNames {
			ln, ok := filter[name]
			if !ok {
				ln = listenerName{RequestedNames: sets.New[string]()}
			}
			ln.RequestedNames.Insert(requestedName)

			// only build the portmap if we aren't filtering this name yet, or if the existing filter is non-empty
			if hasPort && (!ok || len(ln.Ports) != 0) {
				if ln.Ports == nil {
					ln.Ports = map[string]struct{}{}
				}
				ln.Ports.Insert(port)
			} else if !hasPort {
				// if we didn't have a port, we should clear the portmap
				ln.Ports = nil
			}
			filter[name] = ln
		}
	}
	return filter
}

func (g *GrpcConfigGenerator) BuildListeners(node *model.Proxy, push *model.PushContext, names []string) model.Resources {
	filter := newListenerNameFilter(names, node)
	resp := make(model.Resources, 0, len(filter))
	resp = append(resp, buildOutboundListeners(node, push, filter)...)
	resp = append(resp, buildInboundListeners(node, push, filter.inboundNames())...)

	return resp
}

func buildOutboundListeners(node *model.Proxy, push *model.PushContext, filter listenerNames) model.Resources {
	out := make(model.Resources, 0, len(filter))
	// TODO SidecarScope？
	return out
}

func (f listenerNames) inboundNames() []string {
	var out []string
	for key := range f {
		if strings.HasPrefix(key, grpcxds.ServerListenerNamePrefix) {
			out = append(out, key)
		}
	}
	return out
}

func buildInboundListeners(node *model.Proxy, push *model.PushContext, names []string) model.Resources {
	if len(names) == 0 {
		return nil
	}
	var out model.Resources
	// TODO NewMtlsPolicy
	serviceInstancesByPort := map[uint32]model.ServiceTarget{}
	for _, si := range node.ServiceTargets {
		serviceInstancesByPort[si.Port.TargetPort] = si
	}

	for _, name := range names {
		listenAddress := strings.TrimPrefix(name, grpcxds.ServerListenerNamePrefix)
		listenHost, listenPortStr, err := net.SplitHostPort(listenAddress)
		if err != nil {
			klog.Errorf("failed parsing address from gRPC listener name %s: %v", name, err)
			continue
		}
		listenPort, err := strconv.Atoi(listenPortStr)
		if err != nil {
			klog.Errorf("failed parsing port from gRPC listener name %s: %v", name, err)
			continue
		}
		si, ok := serviceInstancesByPort[uint32(listenPort)]
		if !ok {
			klog.Warningf("%s has no service instance for port %s", node.ID, listenPortStr)
			continue
		}

		ll := &listener.Listener{
			Name: name,
			Address: &core.Address{Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Address: listenHost,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: uint32(listenPort),
					},
				},
			}},
			FilterChains: nil,
			// the following must not be set or the client will NACK
			ListenerFilters: nil,
			UseOriginalDst:  nil,
		}
		// add extra addresses for the listener
		extrAddresses := si.Service.GetExtraAddressesForProxy(node)
		if len(extrAddresses) > 0 {
			ll.AdditionalAddresses = util.BuildAdditionalAddresses(extrAddresses, uint32(listenPort))
		}

		out = append(out, &discovery.Resource{
			Name:     ll.Name,
			Resource: protoconv.MessageToAny(ll),
		})
	}
	return out
}

func tryFindFQDN(name string, node *model.Proxy) string {
	// no "." - assuming this is a shortname "foo" -> "foo.ns.svc.cluster.local"
	if !strings.Contains(name, ".") {
		return fmt.Sprintf("%s.%s", name, node.DNSDomain)
	}
	for _, suffix := range []string{
		node.Metadata.Namespace,
		node.Metadata.Namespace + ".svc",
	} {
		shortname := strings.TrimSuffix(name, "."+suffix)
		if shortname != name && strings.HasPrefix(node.DNSDomain, suffix) {
			return fmt.Sprintf("%s.%s", shortname, node.DNSDomain)
		}
	}
	return ""
}
