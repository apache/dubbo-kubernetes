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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
	core "github.com/dubml/xds-api/core/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// CertMaterial holds either in-memory PEM bytes (from a Kubernetes dubbo-xds-* Secret)
// or file paths (from /etc/dubbo/proxy mounted by triple-engine).
type CertMaterial struct {
	CertChainPEM []byte
	KeyPEM       []byte
	RootCertPEM  []byte

	CertChainFile string
	KeyFile       string
	RootCertFile  string
}

// Config configures a Triple xDS ADS client connection to dubbod.
// It can be populated manually in code, automatically via env/args, or directly
// from a Pod's dubbo-xds-* Secret managed by dubbod.
type Config struct {
	BootstrapFile string
	ServerURI     string
	ServerName    string
	Node          *core.Node
	NodeID        string
	Namespace     string
	ClusterID     string
	Generator     string
	Certs         CertMaterial
	Insecure      bool
	Listeners     []string
}

type rawBootstrap struct {
	XDSServers []struct {
		ServerURI    string `json:"server_uri"`
		ChannelCreds []struct {
			Type   string `json:"type"`
			Config struct {
				CertificateFile   string `json:"certificate_file"`
				PrivateKeyFile    string `json:"private_key_file"`
				CACertificateFile string `json:"ca_certificate_file"`
			} `json:"config"`
		} `json:"channel_creds"`
	} `json:"xds_servers"`
	Node          json.RawMessage `json:"node"`
	CertProviders map[string]struct {
		PluginName string `json:"plugin_name"`
		Config     struct {
			CertificateFile   string `json:"certificate_file"`
			PrivateKeyFile    string `json:"private_key_file"`
			CACertificateFile string `json:"ca_certificate_file"`
		} `json:"config"`
	} `json:"certificate_providers"`
}

// ParseBootstrapBytes parses a dubbod triple-bootstrap.json payload into Config.
func ParseBootstrapBytes(data []byte) (Config, error) {
	var raw rawBootstrap
	if err := json.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("parse triple bootstrap json: %w", err)
	}
	if len(raw.XDSServers) == 0 || strings.TrimSpace(raw.XDSServers[0].ServerURI) == "" {
		return Config{}, fmt.Errorf("triple bootstrap missing xds_servers[0].server_uri")
	}

	cfg := Config{
		ServerURI: raw.XDSServers[0].ServerURI,
	}

	if len(raw.Node) > 0 && string(raw.Node) != "null" {
		node := &core.Node{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw.Node, node); err != nil {
			return Config{}, fmt.Errorf("parse bootstrap node: %w", err)
		}
		cfg.Node = node
		cfg.NodeID = node.GetId()
		if fields := node.GetMetadata().GetFields(); fields != nil {
			if v, ok := fields["GENERATOR"]; ok {
				cfg.Generator = v.GetStringValue()
			}
			if v, ok := fields["NAMESPACE"]; ok {
				cfg.Namespace = v.GetStringValue()
			}
			if v, ok := fields["CLUSTER_ID"]; ok {
				cfg.ClusterID = v.GetStringValue()
			}
		}
	}

	// Extract TLS file paths from channel_creds or certificate_providers["default"].
	for _, cred := range raw.XDSServers[0].ChannelCreds {
		switch cred.Type {
		case "insecure":
			cfg.Insecure = true
		case "tls":
			cfg.Certs.CertChainFile = cred.Config.CertificateFile
			cfg.Certs.KeyFile = cred.Config.PrivateKeyFile
			cfg.Certs.RootCertFile = cred.Config.CACertificateFile
		}
	}
	if cfg.Certs.CertChainFile == "" {
		if def, ok := raw.CertProviders["default"]; ok {
			cfg.Certs.CertChainFile = def.Config.CertificateFile
			cfg.Certs.KeyFile = def.Config.PrivateKeyFile
			cfg.Certs.RootCertFile = def.Config.CACertificateFile
		}
	}
	if cfg.Generator == "" {
		cfg.Generator = "triple"
	}
	return cfg, nil
}

// ConfigFromSecretData builds an authenticated Config from a Pod's dubbo-xds-* Secret data.
func ConfigFromSecretData(data map[string][]byte) (Config, error) {
	bootstrapBytes := data[inject.InherentTripleBootstrapFileName]
	if len(bootstrapBytes) == 0 {
		return Config{}, fmt.Errorf("secret missing %s", inject.InherentTripleBootstrapFileName)
	}
	cfg, err := ParseBootstrapBytes(bootstrapBytes)
	if err != nil {
		return Config{}, err
	}
	cfg.Certs.CertChainPEM = append([]byte(nil), data[constants.CertChainFilename]...)
	cfg.Certs.KeyPEM = append([]byte(nil), data[constants.KeyFilename]...)
	cfg.Certs.RootCertPEM = append([]byte(nil), data[constants.CACertNamespaceConfigMapDataName]...)
	if !cfg.Insecure && (len(cfg.Certs.CertChainPEM) == 0 || len(cfg.Certs.KeyPEM) == 0 || len(cfg.Certs.RootCertPEM) == 0) {
		return Config{}, fmt.Errorf("secret missing workload TLS credentials (%s/%s/%s)",
			constants.CertChainFilename, constants.KeyFilename, constants.CACertNamespaceConfigMapDataName)
	}
	return cfg, nil
}

// ResolveConfig merges configuration with precedence: args > env > manual.
func ResolveConfig(manual Config, args []string) (Config, error) {
	cfg := manual

	// Env fallback aligned with webhook injection.
	if cfg.BootstrapFile == "" {
		cfg.BootstrapFile = os.Getenv(inject.InherentTripleBootstrapEnvName)
	}
	if cfg.ServerURI == "" {
		if addr := os.Getenv("XDS_ADDRESS"); addr != "" {
			cfg.ServerURI = addr
		} else if reg := os.Getenv("DUBBO_REGISTRY_ADDRESS"); reg != "" {
			cfg.ServerURI = strings.TrimPrefix(reg, "xds://")
		}
	}
	if cfg.Namespace == "" {
		cfg.Namespace = os.Getenv("POD_NAMESPACE")
	}
	if cfg.ClusterID == "" {
		cfg.ClusterID = os.Getenv("DUBBO_META_CLUSTER_ID")
	}
	if cfg.Generator == "" {
		cfg.Generator = os.Getenv("DUBBO_META_GENERATOR")
	}

	// CLI args override both manual and env.
	if len(args) > 0 {
		fs := flag.NewFlagSet("xdsclient", flag.ContinueOnError)
		var fBootstrap, fAddr, fCertDir, fNodeID, fNS, fGen string
		fs.StringVar(&fBootstrap, "xds-bootstrap", "", "path to triple-bootstrap.json")
		fs.StringVar(&fAddr, "xds-address", "", "dubbod xDS server address")
		fs.StringVar(&fCertDir, "xds-cert-dir", "", "directory containing workload certs")
		fs.StringVar(&fNodeID, "xds-node-id", "", "xDS node identifier")
		fs.StringVar(&fNS, "xds-namespace", "", "workload namespace")
		fs.StringVar(&fGen, "xds-generator", "", "xDS generator name")
		if err := fs.Parse(args); err != nil {
			return Config{}, err
		}
		if fBootstrap != "" {
			cfg.BootstrapFile = fBootstrap
		}
		if fAddr != "" {
			cfg.ServerURI = fAddr
		}
		if fCertDir != "" {
			cfg.Certs.CertChainFile = filepath.Join(fCertDir, constants.CertChainFilename)
			cfg.Certs.KeyFile = filepath.Join(fCertDir, constants.KeyFilename)
			cfg.Certs.RootCertFile = filepath.Join(fCertDir, constants.CACertNamespaceConfigMapDataName)
		}
		if fNodeID != "" {
			cfg.NodeID = fNodeID
		}
		if fNS != "" {
			cfg.Namespace = fNS
		}
		if fGen != "" {
			cfg.Generator = fGen
		}
	}

	if cfg.BootstrapFile != "" {
		raw, err := os.ReadFile(cfg.BootstrapFile)
		if err != nil {
			return Config{}, fmt.Errorf("read bootstrap file %s: %w", cfg.BootstrapFile, err)
		}
		parsed, err := ParseBootstrapBytes(raw)
		if err != nil {
			return Config{}, err
		}
		if cfg.ServerURI == "" {
			cfg.ServerURI = parsed.ServerURI
		}
		if cfg.Node == nil {
			cfg.Node = parsed.Node
		}
		if cfg.NodeID == "" {
			cfg.NodeID = parsed.NodeID
		}
		if cfg.Namespace == "" {
			cfg.Namespace = parsed.Namespace
		}
		if cfg.ClusterID == "" {
			cfg.ClusterID = parsed.ClusterID
		}
		if cfg.Generator == "" {
			cfg.Generator = parsed.Generator
		}
		if cfg.Certs.CertChainFile == "" && len(cfg.Certs.CertChainPEM) == 0 {
			cfg.Certs = parsed.Certs
		}
		if !cfg.Insecure {
			cfg.Insecure = parsed.Insecure
		}
	}

	if cfg.Generator == "" {
		cfg.Generator = "triple"
	}
	if cfg.ClusterID == "" {
		cfg.ClusterID = constants.DefaultClusterName
	}
	if cfg.ServerURI == "" {
		return Config{}, fmt.Errorf("missing xDS server URI")
	}
	node, err := cfg.EffectiveNode()
	if err != nil {
		return Config{}, err
	}
	cfg.Node = node
	return cfg, nil
}

// EffectiveNode builds or normalizes the core.Node sent on ADS streams.
func (c *Config) EffectiveNode() (*core.Node, error) {
	if c.Node != nil && c.Node.GetId() != "" {
		return c.Node, nil
	}
	nodeID := c.NodeID
	if nodeID == "" {
		ns := c.Namespace
		if ns == "" {
			ns = "default"
		}
		host, _ := os.Hostname()
		if host == "" {
			host = "workload"
		}
		ip := os.Getenv("INSTANCE_IP")
		if ip == "" {
			ip = "0.0.0.0"
		}
		nodeID = fmt.Sprintf("inherent~%s~%s.%s~%s.svc.%s", ip, host, ns, ns, constants.DefaultClusterLocalDomain)
	}
	gen := c.Generator
	if gen == "" {
		gen = "triple"
	}
	clusterID := c.ClusterID
	if clusterID == "" {
		clusterID = constants.DefaultClusterName
	}
	metaMap := map[string]any{
		"GENERATOR":  gen,
		"CLUSTER_ID": clusterID,
	}
	if c.Namespace != "" {
		metaMap["NAMESPACE"] = c.Namespace
	}
	meta, err := structpb.NewStruct(metaMap)
	if err != nil {
		return nil, err
	}
	return &core.Node{
		Id:       nodeID,
		Metadata: meta,
	}, nil
}

// BuildClientTLSConfig builds a TLS config from CertMaterial (in-memory PEM or files).
func (m CertMaterial) BuildClientTLSConfig(serverURI, overrideServerName string) (*tls.Config, error) {
	rootPEM := m.RootCertPEM
	if len(rootPEM) == 0 && m.RootCertFile != "" {
		var err error
		rootPEM, err = os.ReadFile(m.RootCertFile)
		if err != nil {
			return nil, fmt.Errorf("read root CA %s: %w", m.RootCertFile, err)
		}
	}
	if len(rootPEM) == 0 {
		return nil, fmt.Errorf("missing root CA certificate")
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(rootPEM) {
		return nil, fmt.Errorf("failed to parse root CA PEM")
	}

	serverName := overrideServerName
	if serverName == "" {
		target := strings.TrimPrefix(strings.TrimPrefix(serverURI, "https://"), "http://")
		if host, _, err := net.SplitHostPort(target); err == nil {
			serverName = host
		} else {
			serverName = target
		}
		if serverName == "127.0.0.1" || serverName == "localhost" || serverName == "::1" {
			serverName = "dubbod.dubbo-system.svc"
		}
	}

	certCopy := m
	return &tls.Config{
		RootCAs:    pool,
		ServerName: serverName,
		MinVersion: tls.VersionTLS12,
		GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			cert, err := certCopy.LoadKeyPair()
			if err != nil {
				return nil, err
			}
			return &cert, nil
		},
	}, nil
}

// LoadKeyPair loads the workload certificate and private key from memory or disk.
func (m CertMaterial) LoadKeyPair() (tls.Certificate, error) {
	if len(m.CertChainPEM) > 0 && len(m.KeyPEM) > 0 {
		return tls.X509KeyPair(m.CertChainPEM, m.KeyPEM)
	}
	if m.CertChainFile != "" && m.KeyFile != "" {
		return tls.LoadX509KeyPair(m.CertChainFile, m.KeyFile)
	}
	return tls.Certificate{}, fmt.Errorf("missing workload certificate and key")
}

// RootCertPool parses the mesh root CA into an x509.CertPool.
func (m CertMaterial) RootCertPool() (*x509.CertPool, error) {
	rootPEM := m.RootCertPEM
	if len(rootPEM) == 0 && m.RootCertFile != "" {
		var err error
		rootPEM, err = os.ReadFile(m.RootCertFile)
		if err != nil {
			return nil, err
		}
	}
	pool := x509.NewCertPool()
	if len(rootPEM) == 0 || !pool.AppendCertsFromPEM(rootPEM) {
		return nil, fmt.Errorf("invalid root certificate PEM")
	}
	return pool, nil
}
