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

package cni

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os/exec"
)

const (
	meshInboundChain = "DUBBO-GRPC-INBOUND"
	meshPodIPSet     = "DUBBO-GRPC-INBOUND-PODS"
)

type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type execCommandRunner struct{}

func (execCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

type IPTablesRuleManager struct {
	path            string
	ipsetPath       string
	grpcInboundPort int
	dryRun          bool
	runner          CommandRunner
}

func NewIPTablesRuleManager(conf NetConf) *IPTablesRuleManager {
	return &IPTablesRuleManager{
		path:            conf.IPTablesPath,
		ipsetPath:       conf.IPSetPath,
		grpcInboundPort: conf.GRPCInboundPort,
		dryRun:          conf.DryRun,
		runner:          execCommandRunner{},
	}
}

func NewIPTablesRuleManagerWithRunner(conf NetConf, runner CommandRunner) *IPTablesRuleManager {
	manager := NewIPTablesRuleManager(conf)
	manager.runner = runner
	return manager
}

func (m *IPTablesRuleManager) AddPodRules(ctx context.Context, podIP string) error {
	ip, err := normalizePodIP(podIP)
	if err != nil {
		return err
	}
	if err := m.ensureBase(ctx); err != nil {
		return err
	}
	return m.runIPSet(ctx, "add", meshPodIPSet, ip, "-exist")
}

func (m *IPTablesRuleManager) DeletePodRules(ctx context.Context, podIP string) error {
	ip, err := normalizePodIP(podIP)
	if err != nil {
		return err
	}
	return m.runIPSet(ctx, "del", meshPodIPSet, ip, "-exist")
}

func (m *IPTablesRuleManager) ensureBase(ctx context.Context) error {
	if err := m.runIPSet(ctx, "create", meshPodIPSet, "hash:ip", "-exist"); err != nil {
		return err
	}
	if err := m.runIgnoreExists(ctx, "-w", "-t", "filter", "-N", meshInboundChain); err != nil {
		return err
	}
	for _, chain := range []string{"FORWARD", "OUTPUT"} {
		if err := m.run(ctx, "-w", "-t", "filter", "-C", chain, "-j", meshInboundChain); err == nil {
			continue
		}
		if err := m.run(ctx, "-w", "-t", "filter", "-I", chain, "1", "-j", meshInboundChain); err != nil {
			return err
		}
	}
	allowGRPCInbound := []string{"-m", "set", "--match-set", meshPodIPSet, "dst", "-p", "tcp", "--dport", fmt.Sprint(m.grpcInboundPort), "-j", "RETURN"}
	rejectOtherTCP := []string{"-m", "set", "--match-set", meshPodIPSet, "dst", "-p", "tcp", "-j", "REJECT"}
	m.deleteRepeated(ctx, allowGRPCInbound...)
	m.deleteRepeated(ctx, rejectOtherTCP...)
	if err := m.appendRule(ctx, allowGRPCInbound...); err != nil {
		return err
	}
	return m.appendRule(ctx, rejectOtherTCP...)
}

func (m *IPTablesRuleManager) deleteRepeated(ctx context.Context, args ...string) {
	cmd := append([]string{"-w", "-t", "filter", "-D", meshInboundChain}, args...)
	for i := 0; i < 20; i++ {
		if err := m.run(ctx, cmd...); err != nil {
			return
		}
	}
}

func (m *IPTablesRuleManager) appendRule(ctx context.Context, args ...string) error {
	cmd := append([]string{"-w", "-t", "filter", "-A", meshInboundChain}, args...)
	return m.run(ctx, cmd...)
}

func (m *IPTablesRuleManager) runIgnoreExists(ctx context.Context, args ...string) error {
	err := m.run(ctx, args...)
	if err == nil {
		return nil
	}
	if stringsContains(err.Error(), "Chain already exists") {
		return nil
	}
	return err
}

func (m *IPTablesRuleManager) run(ctx context.Context, args ...string) error {
	if m.dryRun {
		return nil
	}
	out, err := m.runner.Run(ctx, m.path, args...)
	if err != nil {
		return fmt.Errorf("%s %v: %w: %s", m.path, args, err, string(bytes.TrimSpace(out)))
	}
	return nil
}

func (m *IPTablesRuleManager) runIPSet(ctx context.Context, args ...string) error {
	if m.dryRun {
		return nil
	}
	out, err := m.runner.Run(ctx, m.ipsetPath, args...)
	if err != nil {
		return fmt.Errorf("%s %v: %w: %s", m.ipsetPath, args, err, string(bytes.TrimSpace(out)))
	}
	return nil
}

func normalizePodIP(podIP string) (string, error) {
	ip := net.ParseIP(podIP)
	if ip == nil {
		return "", fmt.Errorf("invalid pod IP %q", podIP)
	}
	if ip.To4() == nil {
		return "", fmt.Errorf("IPv6 pod IP %q is not supported yet", podIP)
	}
	return ip.String(), nil
}

func stringsContains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
