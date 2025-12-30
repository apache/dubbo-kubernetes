//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

import (
	"cmp"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/features"
	"github.com/apache/dubbo-kubernetes/pkg/cluster"
	"github.com/apache/dubbo-kubernetes/pkg/network"
	"github.com/apache/dubbo-kubernetes/pkg/slices"
	dubbomultierror "github.com/apache/dubbo-kubernetes/pkg/util/multierror"
	netutil "github.com/apache/dubbo-kubernetes/pkg/util/net"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/hashicorp/go-multierror"
	"github.com/miekg/dns"
	"k8s.io/apimachinery/pkg/types"
)

type NetworkGatewaysWatcher interface {
	NetworkGateways() []NetworkGateway
	AppendNetworkGatewayHandler(h func())
}

type networkAndCluster struct {
	network network.ID
	cluster cluster.ID
}

type NetworkGatewaysHandler struct {
	handlers []func()
}

type nameCacheEntry struct {
	value  []string
	expiry time.Time
	timer  *time.Timer
}

type networkGatewayNameCache struct {
	NetworkGatewaysHandler
	client *dnsClient

	sync.Mutex
	cache map[string]nameCacheEntry
}

type NetworkGateway struct {
	Network        network.ID
	Cluster        cluster.ID
	Addr           string
	Port           uint32
	HBONEPort      uint32
	ServiceAccount types.NamespacedName
}

type NetworkGateways struct {
	mu                  *sync.RWMutex
	lcm                 uint32
	byNetwork           map[network.ID][]NetworkGateway
	byNetworkAndCluster map[networkAndCluster][]NetworkGateway
}

type NetworkManager struct {
	env        *Environment
	xdsUpdater XDSUpdater
	NameCache  *networkGatewayNameCache

	mu sync.RWMutex

	*NetworkGateways
	Unresolved *NetworkGateways
}

func NewNetworkManager(env *Environment, xdsUpdater XDSUpdater) (*NetworkManager, error) {
	nameCache, err := newNetworkGatewayNameCache()
	if err != nil {
		return nil, err
	}
	mgr := &NetworkManager{
		env:             env,
		NameCache:       nameCache,
		xdsUpdater:      xdsUpdater,
		NetworkGateways: &NetworkGateways{},
		Unresolved:      &NetworkGateways{},
	}

	mgr.NetworkGateways.mu = &mgr.mu
	mgr.Unresolved.mu = &mgr.mu

	// register to per registry, will be called when gateway service changed
	env.AppendNetworkGatewayHandler(mgr.reloadGateways)
	nameCache.AppendNetworkGatewayHandler(mgr.reloadGateways)
	mgr.reload()

	return mgr, nil
}
func newNetworkGatewayNameCacheWithClient(c *dnsClient) *networkGatewayNameCache {
	return &networkGatewayNameCache{client: c, cache: map[string]nameCacheEntry{}}
}

func newNetworkGatewayNameCache() (*networkGatewayNameCache, error) {
	c, err := newClient()
	if err != nil {
		return nil, err
	}
	return newNetworkGatewayNameCacheWithClient(c), nil
}

type dnsClient struct {
	*dns.Client
	resolvConfServers []string
}

func newClient() (*dnsClient, error) {
	c := &dnsClient{
		Client: &dns.Client{
			DialTimeout:  5 * time.Second,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		},
	}
	return c, nil
}

func (mgr *NetworkManager) reloadGateways() {
	changed := mgr.reload()

	if changed && mgr.xdsUpdater != nil {
		log.Infof("gateways changed, triggering push")
		mgr.xdsUpdater.ConfigUpdate(&PushRequest{Full: true, Reason: NewReasonStats(NetworksTrigger), Forced: true})
	}
}

type NetworkGatewaySet = sets.Set[NetworkGateway]

func (mgr *NetworkManager) reload() bool {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	log.Infof("reloading network gateways")

	gatewaySet := make(NetworkGatewaySet)

	gatewaySet.InsertAll(mgr.env.NetworkGateways()...)
	resolvedGatewaySet := mgr.resolveHostnameGateways(gatewaySet)

	return mgr.NetworkGateways.update(resolvedGatewaySet) || mgr.Unresolved.update(gatewaySet)
}

func (mgr *NetworkManager) resolveHostnameGateways(gatewaySet NetworkGatewaySet) NetworkGatewaySet {
	resolvedGatewaySet := make(NetworkGatewaySet, len(gatewaySet))
	// filter the list of gateways to resolve
	hostnameGateways := map[string][]NetworkGateway{}
	names := sets.New[string]()
	for gw := range gatewaySet {
		if netutil.IsValidIPAddress(gw.Addr) {
			resolvedGatewaySet.Insert(gw)
			continue
		}
		if !features.ResolveHostnameGateways {
			log.Warnf("Failed parsing gateway address %s from Service Registry. "+
				"Set RESOLVE_HOSTNAME_GATEWAYS on dubbod to enable resolving hostnames in the control plane.",
				gw.Addr)
			continue
		}
		hostnameGateways[gw.Addr] = append(hostnameGateways[gw.Addr], gw)
		names.Insert(gw.Addr)
	}

	if !features.ResolveHostnameGateways {
		return resolvedGatewaySet
	}
	for host, addrs := range mgr.NameCache.Resolve(names) {
		gwsForHost := hostnameGateways[host]
		if len(addrs) == 0 {
			log.Warnf("could not resolve hostname %q for %d gateways", host, len(gwsForHost))
		}
		for _, gw := range gwsForHost {
			for _, resolved := range addrs {
				// copy the base gateway to preserve the port/network, but update with the resolved IP
				resolvedGw := gw
				resolvedGw.Addr = resolved
				resolvedGatewaySet.Insert(resolvedGw)
			}
		}
	}
	return resolvedGatewaySet
}

func SortGateways(gws []NetworkGateway) []NetworkGateway {
	return slices.SortFunc(gws, func(a, b NetworkGateway) int {
		if r := cmp.Compare(a.Addr, b.Addr); r != 0 {
			return r
		}
		return cmp.Compare(a.Port, b.Port)
	})
}

func (gws *NetworkGateways) allGateways() []NetworkGateway {
	if gws.byNetwork == nil {
		return nil
	}
	out := make([]NetworkGateway, 0)
	for _, gateways := range gws.byNetwork {
		out = append(out, gateways...)
	}
	return SortGateways(out)
}

func networkAndClusterForGateway(g *NetworkGateway) networkAndCluster {
	return networkAndClusterFor(g.Network, g.Cluster)
}

func networkAndClusterFor(nw network.ID, c cluster.ID) networkAndCluster {
	return networkAndCluster{
		network: nw,
		cluster: c,
	}
}

func gcd(x, y int) int {
	var tmp int
	for {
		tmp = x % y
		if tmp > 0 {
			x = y
			y = tmp
		} else {
			return y
		}
	}
}

func lcm(x, y int) int {
	return x * y / gcd(x, y)
}

func (gws *NetworkGateways) update(gatewaySet NetworkGatewaySet) bool {
	if gatewaySet.Equals(sets.New(gws.allGateways()...)) {
		return false
	}

	// index by network or network+cluster for quick lookup
	byNetwork := make(map[network.ID][]NetworkGateway)
	byNetworkAndCluster := make(map[networkAndCluster][]NetworkGateway)
	for gw := range gatewaySet {
		byNetwork[gw.Network] = append(byNetwork[gw.Network], gw)
		nc := networkAndClusterForGateway(&gw)
		byNetworkAndCluster[nc] = append(byNetworkAndCluster[nc], gw)
	}

	var gwNum []int
	// Sort the gateways in byNetwork, and also calculate the max number
	// of gateways per network.
	for k, gws := range byNetwork {
		byNetwork[k] = SortGateways(gws)
		gwNum = append(gwNum, len(gws))
	}

	// Sort the gateways in byNetworkAndCluster.
	for k, gws := range byNetworkAndCluster {
		byNetworkAndCluster[k] = SortGateways(gws)
		gwNum = append(gwNum, len(gws))
	}

	lcmVal := 1
	// calculate lcm
	for _, num := range gwNum {
		lcmVal = lcm(lcmVal, num)
	}

	gws.lcm = uint32(lcmVal)
	gws.byNetwork = byNetwork
	gws.byNetworkAndCluster = byNetworkAndCluster

	return true
}

func (ngh *NetworkGatewaysHandler) AppendNetworkGatewayHandler(h func()) {
	ngh.handlers = append(ngh.handlers, h)
}

func (n *networkGatewayNameCache) Resolve(names sets.String) map[string][]string {
	n.Lock()
	defer n.Unlock()

	n.cleanupWatches(names)

	out := make(map[string][]string, len(names))
	for name := range names {
		out[name] = n.resolveFromCache(name)
	}

	return out
}

func (n *networkGatewayNameCache) cleanupWatches(names sets.String) {
	for name, entry := range n.cache {
		if names.Contains(name) {
			continue
		}
		entry.timer.Stop()
		delete(n.cache, name)
	}
}

var (
	// MinGatewayTTL is exported for testing
	MinGatewayTTL = 30 * time.Second

	// https://github.com/coredns/coredns/blob/v1.10.1/plugin/pkg/dnsutil/ttl.go#L51
	MaxGatewayTTL = 1 * time.Hour
)

func (n *networkGatewayNameCache) resolveAndCache(name string) []string {
	entry, ok := n.cache[name]
	if ok {
		entry.timer.Stop()
	}
	delete(n.cache, name)
	addrs, ttl, err := n.resolve(name)
	// avoid excessive pushes due to small TTL
	if ttl < MinGatewayTTL {
		ttl = MinGatewayTTL
	}
	expiry := time.Now().Add(ttl)
	if err != nil {
		// gracefully retain old addresses in case the DNS server is unavailable
		addrs = entry.value
	}
	n.cache[name] = nameCacheEntry{
		value:  addrs,
		expiry: expiry,
		// TTL expires, try to refresh TODO should this be < ttl?
		timer: time.AfterFunc(ttl, n.refreshAndNotify(name)),
	}

	return addrs
}

type MsgHdr struct {
	Id                 uint16
	Response           bool
	Opcode             int
	Authoritative      bool
	Truncated          bool
	RecursionDesired   bool
	RecursionAvailable bool
	Zero               bool
	AuthenticatedData  bool
	CheckingDisabled   bool
	Rcode              int
}

func (n *networkGatewayNameCache) resolve(name string) ([]string, time.Duration, error) {
	ttl := MaxGatewayTTL
	var out []string
	errs := dubbomultierror.New()

	var mu sync.Mutex
	var wg sync.WaitGroup
	doResolve := func(dnsType uint16) {
		defer wg.Done()

		res := n.client.Query(new(dns.Msg).SetQuestion(dns.Fqdn(name), dnsType))

		mu.Lock()
		defer mu.Unlock()
		if res.Rcode == dns.RcodeServerFailure {
			errs = multierror.Append(errs, fmt.Errorf("upstream dns failure, qtype: %v", dnsType))
			return
		}
		for _, rr := range res.Answer {
			switch record := rr.(type) {
			case *dns.A:
				out = append(out, record.A.String())
			case *dns.AAAA:
				out = append(out, record.AAAA.String())
			}
		}
		if nextTTL := minimalTTL(res); nextTTL < ttl {
			ttl = nextTTL
		}
	}

	wg.Add(2)
	go doResolve(dns.TypeA)
	go doResolve(dns.TypeAAAA)
	wg.Wait()

	sort.Strings(out)
	if errs.Len() == 2 {
		return out, MinGatewayTTL, errs
	}
	return out, ttl, nil
}

func (n *networkGatewayNameCache) resolveFromCache(name string) []string {
	if entry, ok := n.cache[name]; ok && entry.expiry.After(time.Now()) {
		return entry.value
	}
	return n.resolveAndCache(name)
}

func (n *networkGatewayNameCache) refreshAndNotify(name string) func() {
	return func() {
		log.Debugf("network gateways: refreshing DNS for %s", name)
		n.Lock()
		old := n.cache[name]
		addrs := n.resolveAndCache(name)
		n.Unlock()

		if !slices.Equal(old.value, addrs) {
			log.Debugf("network gateways: DNS for %s changed: %v -> %v", name, old.value, addrs)
			n.NotifyGatewayHandlers()
		}
	}
}

func minimalTTL(m *dns.Msg) time.Duration {
	// No records or OPT is the only record, return a short ttl as a fail safe.
	if len(m.Answer)+len(m.Ns) == 0 &&
		(len(m.Extra) == 0 || (len(m.Extra) == 1 && m.Extra[0].Header().Rrtype == dns.TypeOPT)) {
		return MinGatewayTTL
	}

	minTTL := MaxGatewayTTL
	for _, r := range m.Answer {
		if r.Header().Ttl < uint32(minTTL.Seconds()) {
			minTTL = time.Duration(r.Header().Ttl) * time.Second
		}
	}
	for _, r := range m.Ns {
		if r.Header().Ttl < uint32(minTTL.Seconds()) {
			minTTL = time.Duration(r.Header().Ttl) * time.Second
		}
	}

	for _, r := range m.Extra {
		if r.Header().Rrtype == dns.TypeOPT {
			// OPT records use TTL field for extended rcode and flags
			continue
		}
		if r.Header().Ttl < uint32(minTTL.Seconds()) {
			minTTL = time.Duration(r.Header().Ttl) * time.Second
		}
	}
	return minTTL
}

func (ngh *NetworkGatewaysHandler) NotifyGatewayHandlers() {
	for _, handler := range ngh.handlers {
		handler()
	}
}

func (c *dnsClient) Query(req *dns.Msg) *dns.Msg {
	var response *dns.Msg
	for _, upstream := range c.resolvConfServers {
		cResponse, _, err := c.Exchange(req, upstream)
		rcode := dns.RcodeServerFailure
		if err == nil && cResponse != nil {
			rcode = cResponse.Rcode
		}
		if rcode == dns.RcodeServerFailure {
			// RcodeServerFailure means the upstream cannot serve the request
			// https://github.com/coredns/coredns/blob/v1.10.1/plugin/forward/forward.go#L193
			log.Infof("upstream dns failure: %v: %v: %v", upstream, getReqNames(req), err)
			continue
		}
		response = cResponse
		if rcode == dns.RcodeSuccess {
			break
		}
		codeString := dns.RcodeToString[rcode]
		log.Debugf("upstream dns error: %v: %v: %v", upstream, getReqNames(req), codeString)
	}
	if response == nil {
		response = new(dns.Msg)
		response.SetReply(req)
		response.Rcode = dns.RcodeServerFailure
	}
	return response
}

func getReqNames(req *dns.Msg) []string {
	names := make([]string, 0, 1)
	for _, qq := range req.Question {
		names = append(names, qq.Name)
	}
	return names
}
