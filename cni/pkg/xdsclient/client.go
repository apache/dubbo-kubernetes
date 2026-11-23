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

package xdsclient

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	pkgmodel "github.com/apache/dubbo-kubernetes/pkg/model"
	clusterv1 "github.com/dubml/xds-api/cluster/v1"
	core "github.com/dubml/xds-api/core/v1"
	endpointv1 "github.com/dubml/xds-api/endpoint/v1"
	listenerv1 "github.com/dubml/xds-api/listener/v1"
	routev1 "github.com/dubml/xds-api/route/v1"
	discoveryv1 "github.com/dubml/xds-api/service/discovery/v1"
	rpcstatus "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

const (
	initialBackoff = 50 * time.Millisecond
	maxBackoff     = 5 * time.Second
)

// Status describes the live ADS synchronization state of a managed workload client.
type Status struct {
	NodeID           string            `json:"node_id"`
	Connected        bool              `json:"connected"`
	Ready            bool              `json:"ready"`
	Attempts         uint64            `json:"attempts"`
	Generation       uint64            `json:"generation"`
	AcceptedVersions map[string]string `json:"accepted_versions,omitempty"`
	LastError        string            `json:"last_error,omitempty"`
}

type typeState struct {
	names   []string
	version string
	nonce   string
}

// Client manages a persistent authenticated ADS stream to dubbod on behalf of a workload
// and publishes atomic routing Snapshots for the node-level Triple tunnel proxy.
type Client struct {
	mu       sync.RWMutex
	cfg      Config
	cache    *resourceCache
	states   map[string]*typeState
	status   Status
	snapshot atomic.Pointer[Snapshot]

	reconnectCh chan struct{}
}

// New creates a new xDS ADS Client with the validated Config.
func New(cfg Config) (*Client, error) {
	node, err := cfg.EffectiveNode()
	if err != nil {
		return nil, err
	}
	cfg.Node = node

	c := &Client{
		cfg:   cfg,
		cache: newResourceCache(),
		states: map[string]*typeState{
			pkgmodel.ListenerType: {names: append([]string(nil), cfg.Listeners...)},
			pkgmodel.RouteType:    {},
			pkgmodel.ClusterType:  {},
			pkgmodel.EndpointType: {},
		},
		status: Status{
			NodeID:           node.GetId(),
			AcceptedVersions: map[string]string{},
		},
		reconnectCh: make(chan struct{}, 1),
	}
	c.snapshot.Store(&Snapshot{
		Listeners: map[string]*routev1.RouteConfiguration{},
		Clusters:  map[string]*TargetCluster{},
	})
	return c, nil
}

// Snapshot returns the latest atomically published routing snapshot.
func (c *Client) Snapshot() *Snapshot {
	return c.snapshot.Load()
}

// Status returns a copy of the current connection and sync status.
func (c *Client) Status() Status {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := c.status
	out.AcceptedVersions = make(map[string]string, len(c.status.AcceptedVersions))
	for k, v := range c.status.AcceptedVersions {
		out.AcceptedVersions[k] = v
	}
	return out
}

// UpdateConfig updates credentials or bootstrap config (e.g. on Secret rotation)
// and triggers an immediate reconnect so dubbod sees the renewed mTLS certificate.
func (c *Client) UpdateConfig(cfg Config) error {
	node, err := cfg.EffectiveNode()
	if err != nil {
		return err
	}
	cfg.Node = node
	c.mu.Lock()
	c.cfg = cfg
	c.status.NodeID = node.GetId()
	c.mu.Unlock()
	select {
	case c.reconnectCh <- struct{}{}:
	default:
	}
	return nil
}

// Run maintains the ADS stream until ctx is cancelled, reconnecting with bounded backoff.
func (c *Client) Run(ctx context.Context) error {
	backoff := initialBackoff
	for {
		if ctx.Err() != nil {
			c.setDisconnected("")
			return ctx.Err()
		}

		c.mu.Lock()
		c.status.Attempts++
		c.mu.Unlock()

		err := c.runSession(ctx)
		if ctx.Err() != nil {
			c.setDisconnected("")
			return ctx.Err()
		}

		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		}
		c.setDisconnected(errMsg)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.reconnectCh:
			backoff = initialBackoff
		case <-time.After(backoff):
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

func (c *Client) setDisconnected(errMsg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status.Connected = false
	if errMsg != "" {
		c.status.LastError = errMsg
	}
}

func (c *Client) runSession(parentCtx context.Context) error {
	sessionCtx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	// Cancel active stream immediately when UpdateConfig signals credential rotation.
	go func() {
		select {
		case <-sessionCtx.Done():
		case <-c.reconnectCh:
			cancel()
		}
	}()

	c.mu.RLock()
	cfg := c.cfg
	c.mu.RUnlock()

	dialOpts, err := buildDialOptions(cfg)
	if err != nil {
		return fmt.Errorf("build xDS dial options: %w", err)
	}
	conn, err := grpc.NewClient(cfg.ServerURI, dialOpts...)
	if err != nil {
		return fmt.Errorf("dial xDS server %s: %w", cfg.ServerURI, err)
	}
	defer conn.Close()

	adsClient := discoveryv1.NewAggregatedDiscoveryServiceClient(conn)
	stream, err := adsClient.StreamAggregatedResources(sessionCtx)
	if err != nil {
		return fmt.Errorf("open ADS stream: %w", err)
	}

	// Clear old stream nonces and replay initial LDS + any active subscriptions.
	c.mu.Lock()
	for _, st := range c.states {
		st.nonce = ""
	}
	node := cfg.Node
	initialReqs := []*discoveryv1.DiscoveryRequest{
		{
			Node:          node,
			TypeUrl:       pkgmodel.ListenerType,
			VersionInfo:   c.states[pkgmodel.ListenerType].version,
			ResourceNames: append([]string(nil), c.states[pkgmodel.ListenerType].names...),
		},
	}
	for _, typeURL := range []string{pkgmodel.RouteType, pkgmodel.ClusterType, pkgmodel.EndpointType} {
		st := c.states[typeURL]
		if len(st.names) > 0 {
			initialReqs = append(initialReqs, &discoveryv1.DiscoveryRequest{
				Node:          node,
				TypeUrl:       typeURL,
				VersionInfo:   st.version,
				ResourceNames: append([]string(nil), st.names...),
			})
		}
	}
	c.mu.Unlock()

	for _, req := range initialReqs {
		if err := stream.Send(req); err != nil {
			return fmt.Errorf("send initial xDS request (%s): %w", req.TypeUrl, err)
		}
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			return err
		}
		followups, ackErr := c.handleResponse(node, resp)
		if ackErr != nil {
			// NACK with last accepted version and error detail; keep prior snapshot intact.
			c.mu.RLock()
			lastVer := ""
			var names []string
			if st, ok := c.states[resp.GetTypeUrl()]; ok {
				lastVer = st.version
				names = append([]string(nil), st.names...)
			}
			c.mu.RUnlock()
			nack := &discoveryv1.DiscoveryRequest{
				Node:          node,
				TypeUrl:       resp.GetTypeUrl(),
				VersionInfo:   lastVer,
				ResponseNonce: resp.GetNonce(),
				ResourceNames: names,
				ErrorDetail: &rpcstatus.Status{
					Code:    int32(codes.InvalidArgument),
					Message: ackErr.Error(),
				},
			}
			if sendErr := stream.Send(nack); sendErr != nil {
				return sendErr
			}
			continue
		}
		for _, req := range followups {
			if sendErr := stream.Send(req); sendErr != nil {
				return sendErr
			}
		}
	}
}

func (c *Client) handleResponse(node *core.Node, resp *discoveryv1.DiscoveryResponse) ([]*discoveryv1.DiscoveryRequest, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	typeURL := resp.GetTypeUrl()
	switch typeURL {
	case pkgmodel.ListenerType:
		next := make(map[string]*listenerv1.Listener, len(resp.GetResources()))
		for _, anyRes := range resp.GetResources() {
			l := &listenerv1.Listener{}
			if err := anyRes.UnmarshalTo(l); err != nil {
				return nil, fmt.Errorf("decode LDS resource: %w", err)
			}
			next[l.GetName()] = l
		}
		c.cache.listeners = next
	case pkgmodel.RouteType:
		next := make(map[string]*routev1.RouteConfiguration, len(resp.GetResources()))
		for _, anyRes := range resp.GetResources() {
			rc := &routev1.RouteConfiguration{}
			if err := anyRes.UnmarshalTo(rc); err != nil {
				return nil, fmt.Errorf("decode RDS resource: %w", err)
			}
			next[rc.GetName()] = rc
		}
		c.cache.routes = next
	case pkgmodel.ClusterType:
		next := make(map[string]*clusterv1.Cluster, len(resp.GetResources()))
		for _, anyRes := range resp.GetResources() {
			cl := &clusterv1.Cluster{}
			if err := anyRes.UnmarshalTo(cl); err != nil {
				return nil, fmt.Errorf("decode CDS resource: %w", err)
			}
			next[cl.GetName()] = cl
		}
		c.cache.clusters = next
	case pkgmodel.EndpointType:
		for _, anyRes := range resp.GetResources() {
			cla := &endpointv1.ClusterLoadAssignment{}
			if err := anyRes.UnmarshalTo(cla); err != nil {
				return nil, fmt.Errorf("decode EDS resource: %w", err)
			}
			c.cache.endpoints[cla.GetClusterName()] = cla
		}
	default:
		return nil, fmt.Errorf("unsupported xDS type URL %q", typeURL)
	}

	st, ok := c.states[typeURL]
	if !ok {
		st = &typeState{}
		c.states[typeURL] = st
	}
	st.version = resp.GetVersionInfo()
	st.nonce = resp.GetNonce()

	nextGen := c.status.Generation + 1
	snap, deps := c.cache.compile(nextGen)
	c.snapshot.Store(snap)

	c.status.Connected = true
	c.status.Ready = snap.Ready
	c.status.Generation = nextGen
	c.status.AcceptedVersions[typeURL] = resp.GetVersionInfo()
	c.status.LastError = ""

	// 1) Always ACK the received response so dubbod marks this type SYNCED.
	reqs := []*discoveryv1.DiscoveryRequest{
		{
			Node:          node,
			TypeUrl:       typeURL,
			VersionInfo:   st.version,
			ResponseNonce: st.nonce,
			ResourceNames: append([]string(nil), st.names...),
		},
	}

	// 2) Trigger downstream RDS / CDS / EDS subscriptions when dependencies change.
	depMap := map[string][]string{
		pkgmodel.RouteType:    deps.routes,
		pkgmodel.ClusterType:  deps.clusters,
		pkgmodel.EndpointType: deps.endpoints,
	}
	for _, depType := range []string{pkgmodel.RouteType, pkgmodel.ClusterType, pkgmodel.EndpointType} {
		desired := depMap[depType]
		depState := c.states[depType]
		if len(desired) == 0 || reflect.DeepEqual(depState.names, desired) {
			continue
		}
		depState.names = desired
		reqs = append(reqs, &discoveryv1.DiscoveryRequest{
			Node:          node,
			TypeUrl:       depType,
			VersionInfo:   depState.version,
			ResponseNonce: depState.nonce,
			ResourceNames: append([]string(nil), desired...),
		})
	}
	return reqs, nil
}

func buildDialOptions(cfg Config) ([]grpc.DialOption, error) {
	opts := []grpc.DialOption{
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	}
	if cfg.Insecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		return opts, nil
	}
	tlsCfg, err := cfg.Certs.BuildClientTLSConfig(cfg.ServerURI, cfg.ServerName)
	if err != nil {
		return nil, err
	}
	opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)))
	return opts, nil
}
