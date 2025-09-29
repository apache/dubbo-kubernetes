/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dubboagent

import (
	"fmt"
	dubbogrpc "github.com/apache/dubbo-kubernetes/sail/pkg/grpc"
	"github.com/apache/dubbo-kubernetes/security/pkg/nodeagent/caclient"
	"google.golang.org/grpc"
	meshconfig "istio.io/api/mesh/v1alpha1"
	"net"
	"sync"
)

type XdsProxy struct {
	stopChan             chan struct{}
	downstreamGrpcServer *grpc.Server
	downstreamListener   net.Listener
	optsMutex            sync.RWMutex
	dialOptions          []grpc.DialOption
	dubbodSAN            string
}

func initXdsProxy(ia *Agent) (*XdsProxy, error) {
	proxy := &XdsProxy{}
	return proxy, nil
}

func (p *XdsProxy) initDubbodDialOptions(agent *Agent) error {
	opts, err := p.buildUpstreamClientDialOpts(agent)
	if err != nil {
		return err
	}

	p.optsMutex.Lock()
	p.dialOptions = opts
	p.optsMutex.Unlock()
	return nil
}

func (p *XdsProxy) buildUpstreamClientDialOpts(sa *Agent) ([]grpc.DialOption, error) {
	tlsOpts, err := p.getTLSOptions(sa)
	if err != nil {
		return nil, fmt.Errorf("failed to get TLS options to talk to upstream: %v", err)
	}
	options, err := dubbogrpc.ClientOptions(nil, tlsOpts)
	if err != nil {
		return nil, err
	}
	if sa.secOpts.CredFetcher != nil {
		options = append(options, grpc.WithPerRPCCredentials(caclient.NewDefaultTokenProvider(sa.secOpts)))
	}
	return options, nil
}

func (p *XdsProxy) getTLSOptions(agent *Agent) (*dubbogrpc.TLSOptions, error) {
	if agent.proxyConfig.ControlPlaneAuthPolicy == meshconfig.AuthenticationPolicy_NONE {
		return nil, nil
	}
	xdsCACertPath, err := agent.FindRootCAForXDS()
	if err != nil {
		return nil, fmt.Errorf("failed to find root CA cert for XDS: %v", err)
	}
	key, cert := agent.GetKeyCertsForXDS()
	return &dubbogrpc.TLSOptions{
		RootCert:      xdsCACertPath,
		Key:           key,
		Cert:          cert,
		ServerAddress: agent.proxyConfig.DiscoveryAddress,
		SAN:           p.dubbodSAN,
	}, nil
}

func (p *XdsProxy) close() {
	close(p.stopChan)
	if p.downstreamGrpcServer != nil {
		p.downstreamGrpcServer.Stop()
	}
	if p.downstreamListener != nil {
		_ = p.downstreamListener.Close()
	}
}
