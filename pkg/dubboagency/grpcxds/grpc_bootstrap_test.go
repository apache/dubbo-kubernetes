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

package grpcxds

import (
	"strings"
	"testing"

	pkgmodel "github.com/apache/dubbo-kubernetes/pkg/model"
)

func TestGenerateBootstrapUsesTLSForDirectDiscovery(t *testing.T) {
	bootstrap, err := GenerateBootstrap(GenerateBootstrapOptions{
		Node: &pkgmodel.Node{
			ID:       "proxyless~10.0.0.1~app.default~default.svc.cluster.local",
			Metadata: &pkgmodel.BootstrapNodeMetadata{},
		},
		DiscoveryAddress: "dubbod.dubbo-system.svc:15012",
		CertDir:          "/etc/dubbo/proxy",
	})
	if err != nil {
		t.Fatalf("GenerateBootstrap() failed: %v", err)
	}
	if got, want := bootstrap.XDSServers[0].ChannelCreds[0].Type, "tls"; got != want {
		t.Fatalf("channel_creds type = %q, want %q", got, want)
	}
	if got, want := bootstrap.XDSServers[0].ServerFeatures[0], "xds_v3"; got != want {
		t.Fatalf("server feature = %q, want %q", got, want)
	}

	cfg, ok := bootstrap.XDSServers[0].ChannelCreds[0].Config.(FileWatcherCertProviderConfig)
	if !ok {
		t.Fatalf("tls config type = %T, want FileWatcherCertProviderConfig", bootstrap.XDSServers[0].ChannelCreds[0].Config)
	}
	if got, want := cfg.CACertificateFile, "/etc/dubbo/proxy/root-cert.pem"; got != want {
		t.Fatalf("ca_certificate_file = %q, want %q", got, want)
	}
	if got, want := cfg.CertificateFile, "/etc/dubbo/proxy/cert-chain.pem"; got != want {
		t.Fatalf("certificate_file = %q, want %q", got, want)
	}
	if got, want := cfg.PrivateKeyFile, "/etc/dubbo/proxy/key.pem"; got != want {
		t.Fatalf("private_key_file = %q, want %q", got, want)
	}
}

func TestGenerateBootstrapUsesInsecureForUDS(t *testing.T) {
	bootstrap, err := GenerateBootstrap(GenerateBootstrapOptions{
		Node: &pkgmodel.Node{
			ID:       "proxyless~10.0.0.1~app.default~default.svc.cluster.local",
			Metadata: &pkgmodel.BootstrapNodeMetadata{},
		},
		XdsUdsPath: "/tmp/xds.sock",
		CertDir:    "/etc/dubbo/proxy",
	})
	if err != nil {
		t.Fatalf("GenerateBootstrap() failed: %v", err)
	}
	if !strings.HasPrefix(bootstrap.XDSServers[0].ServerURI, "unix://") {
		t.Fatalf("server_uri = %q, want unix:// prefix", bootstrap.XDSServers[0].ServerURI)
	}
	if got, want := bootstrap.XDSServers[0].ChannelCreds[0].Type, "insecure"; got != want {
		t.Fatalf("channel_creds type = %q, want %q", got, want)
	}
}
