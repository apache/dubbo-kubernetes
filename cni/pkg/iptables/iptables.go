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

package iptables

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	cniconstants "github.com/apache/dubbo-kubernetes/cni/pkg/constants"
)

const (
	ChainInbound    = "DUBBO_INBOUND"
	ChainOutbound   = "DUBBO_OUTPUT"
	ChainInRedirect = "DUBBO_IN_REDIRECT"
	ChainRedirect   = "DUBBO_REDIRECT"
)

// Config defines transparent traffic redirection rules applied inside an enrolled Pod's netns.
type Config struct {
	OutboundPort         int
	InboundPort          int
	InboundTunnelPort    int
	ProxyUID             int
	ExcludeOutboundPorts []int
	ExcludeInboundPorts  []int
	ExcludeOutboundCIDRs []string
	NodeIP               string
}

// DefaultConfig returns the default Route C redirection configuration.
func DefaultConfig() Config {
	return Config{
		OutboundPort:      cniconstants.OutboundCapturePort,
		InboundPort:       cniconstants.InboundPlaintextPort,
		InboundTunnelPort: cniconstants.InboundTunnelPort,
		ProxyUID:          cniconstants.DefaultProxyUID,
		ExcludeOutboundPorts: []int{
			cniconstants.DefaultDubbodXDSPort,
			cniconstants.DefaultStatusPort,
		},
		ExcludeInboundPorts: []int{
			cniconstants.InboundTunnelPort,
			cniconstants.DefaultStatusPort,
		},
	}
}

// BuildRestoreScript renders an idempotent iptables-restore nat table script
// capturing Pod TCP traffic into the node-level Triple tunnel proxy.
func BuildRestoreScript(cfg Config) string {
	if cfg.OutboundPort <= 0 {
		cfg.OutboundPort = cniconstants.OutboundCapturePort
	}
	if cfg.InboundPort <= 0 {
		cfg.InboundPort = cniconstants.InboundPlaintextPort
	}
	if cfg.InboundTunnelPort <= 0 {
		cfg.InboundTunnelPort = cniconstants.InboundTunnelPort
	}

	var b strings.Builder
	b.WriteString("*nat\n")
	fmt.Fprintf(&b, ":%s - [0:0]\n", ChainInbound)
	fmt.Fprintf(&b, ":%s - [0:0]\n", ChainInRedirect)
	fmt.Fprintf(&b, ":%s - [0:0]\n", ChainOutbound)
	fmt.Fprintf(&b, ":%s - [0:0]\n", ChainRedirect)

	// Redirect chains.
	if cfg.NodeIP != "" {
		fmt.Fprintf(&b, "-A %s -p tcp -j DNAT --to-destination %s:%d\n", ChainRedirect, cfg.NodeIP, cfg.OutboundPort)
		fmt.Fprintf(&b, "-A %s -p tcp -j DNAT --to-destination %s:%d\n", ChainInRedirect, cfg.NodeIP, cfg.InboundPort)
	} else {
		fmt.Fprintf(&b, "-A %s -p tcp -j REDIRECT --to-ports %d\n", ChainRedirect, cfg.OutboundPort)
		fmt.Fprintf(&b, "-A %s -p tcp -j REDIRECT --to-ports %d\n", ChainInRedirect, cfg.InboundPort)
	}

	// Inbound hook: skip tunnel port and explicitly excluded ports.
	fmt.Fprintf(&b, "-A PREROUTING -p tcp -j %s\n", ChainInbound)
	for _, port := range dedupePorts(append([]int{cfg.InboundTunnelPort}, cfg.ExcludeInboundPorts...)) {
		fmt.Fprintf(&b, "-A %s -p tcp --dport %d -j RETURN\n", ChainInbound, port)
	}
	fmt.Fprintf(&b, "-A %s -p tcp -j %s\n", ChainInbound, ChainInRedirect)

	// Outbound hook: skip loopback, proxy UID, control-plane xDS port, and excluded CIDRs.
	fmt.Fprintf(&b, "-A OUTPUT -p tcp -j %s\n", ChainOutbound)
	fmt.Fprintf(&b, "-A %s -o lo -j RETURN\n", ChainOutbound)
	fmt.Fprintf(&b, "-A %s -d 127.0.0.1/32 -j RETURN\n", ChainOutbound)
	if cfg.ProxyUID > 0 {
		fmt.Fprintf(&b, "-A %s -m owner --uid-owner %d -j RETURN\n", ChainOutbound, cfg.ProxyUID)
	}
	for _, cidr := range cfg.ExcludeOutboundCIDRs {
		if trimmed := strings.TrimSpace(cidr); trimmed != "" {
			fmt.Fprintf(&b, "-A %s -d %s -j RETURN\n", ChainOutbound, trimmed)
		}
	}
	for _, port := range dedupePorts(cfg.ExcludeOutboundPorts) {
		fmt.Fprintf(&b, "-A %s -p tcp --dport %d -j RETURN\n", ChainOutbound, port)
	}
	fmt.Fprintf(&b, "-A %s -p tcp -j %s\n", ChainOutbound, ChainRedirect)
	b.WriteString("COMMIT\n")
	return b.String()
}

// BuildCleanupScript renders an iptables-restore script removing DUBBO_* hooks.
func BuildCleanupScript() string {
	var b strings.Builder
	b.WriteString("*nat\n")
	fmt.Fprintf(&b, "-D PREROUTING -p tcp -j %s\n", ChainInbound)
	fmt.Fprintf(&b, "-D OUTPUT -p tcp -j %s\n", ChainOutbound)
	fmt.Fprintf(&b, "-F %s\n", ChainInbound)
	fmt.Fprintf(&b, "-F %s\n", ChainInRedirect)
	fmt.Fprintf(&b, "-F %s\n", ChainOutbound)
	fmt.Fprintf(&b, "-F %s\n", ChainRedirect)
	fmt.Fprintf(&b, "-X %s\n", ChainInbound)
	fmt.Fprintf(&b, "-X %s\n", ChainInRedirect)
	fmt.Fprintf(&b, "-X %s\n", ChainOutbound)
	fmt.Fprintf(&b, "-X %s\n", ChainRedirect)
	b.WriteString("COMMIT\n")
	return b.String()
}

// Redirector applies and cleans up per-Pod netns traffic redirection rules.
type Redirector interface {
	Apply(ctx context.Context, netnsPath string, cfg Config) error
	Cleanup(ctx context.Context, netnsPath string) error
}

// CommandRunner executes a host command with stdin (injectable for tests).
type CommandRunner func(ctx context.Context, stdin string, name string, args ...string) ([]byte, error)

// NetNSRedirector applies iptables-restore rules inside a target netns using nsenter.
type NetNSRedirector struct {
	Runner CommandRunner
}

func defaultRunner(ctx context.Context, stdin string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = strings.NewReader(stdin)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.Bytes(), err
}

func (r *NetNSRedirector) run(ctx context.Context, stdin, name string, args ...string) ([]byte, error) {
	if r != nil && r.Runner != nil {
		return r.Runner(ctx, stdin, name, args...)
	}
	return defaultRunner(ctx, stdin, name, args...)
}

// Apply writes the DUBBO_* NAT rules inside netnsPath.
func (r *NetNSRedirector) Apply(ctx context.Context, netnsPath string, cfg Config) error {
	if strings.TrimSpace(netnsPath) == "" {
		return fmt.Errorf("netns path is required")
	}
	script := BuildRestoreScript(cfg)
	out, err := r.run(ctx, script, "nsenter", "--net="+netnsPath, "--", "iptables-restore", "--noflush")
	if err != nil {
		return fmt.Errorf("apply iptables in netns %s: %w (output: %s)", netnsPath, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Cleanup removes the DUBBO_* NAT rules from netnsPath.
func (r *NetNSRedirector) Cleanup(ctx context.Context, netnsPath string) error {
	if strings.TrimSpace(netnsPath) == "" {
		return nil
	}
	script := BuildCleanupScript()
	out, err := r.run(ctx, script, "nsenter", "--net="+netnsPath, "--", "iptables-restore", "--noflush")
	if err != nil {
		return fmt.Errorf("cleanup iptables in netns %s: %w (output: %s)", netnsPath, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func dedupePorts(ports []int) []int {
	seen := make(map[int]struct{}, len(ports))
	out := make([]int, 0, len(ports))
	for _, p := range ports {
		if p <= 0 || p > 65535 {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

// FormatPortList formats ports as a comma-separated list.
func FormatPortList(ports []int) string {
	deduped := dedupePorts(ports)
	parts := make([]string, len(deduped))
	for i, p := range deduped {
		parts[i] = strconv.Itoa(p)
	}
	return strings.Join(parts, ",")
}
