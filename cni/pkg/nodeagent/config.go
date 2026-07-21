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

package nodeagent

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
)

const (
	defaultStateDir     = "/var/run/dubbo-cni"
	defaultIPTablesPath = "iptables"
	defaultIPSetPath    = "ipset"
	PluginType          = "dubbo-cni"
)

type NetConf struct {
	CNIVersion        string          `json:"cniVersion,omitempty"`
	Name              string          `json:"name,omitempty"`
	Type              string          `json:"type,omitempty"`
	PrevResult        json.RawMessage `json:"prevResult,omitempty"`
	Kubernetes        KubernetesConf  `json:"kubernetes,omitempty"`
	KubeConfig        string          `json:"kubeconfig,omitempty"`
	ManagedLabel      string          `json:"managedLabel,omitempty"`
	ManagedLabelValue string          `json:"managedLabelValue,omitempty"`
	GRPCInboundPort   int             `json:"grpcInboundPort,omitempty"`
	StateDir          string          `json:"stateDir,omitempty"`
	IPTablesPath      string          `json:"iptablesPath,omitempty"`
	IPSetPath         string          `json:"ipsetPath,omitempty"`
	DryRun            bool            `json:"dryRun,omitempty"`
}

type KubernetesConf struct {
	KubeConfig string `json:"kubeconfig,omitempty"`
}

func ParseNetConf(data []byte) (NetConf, error) {
	conf := NetConf{}
	if len(strings.TrimSpace(string(data))) > 0 {
		if err := json.Unmarshal(data, &conf); err != nil {
			return conf, fmt.Errorf("parse CNI config: %w", err)
		}
	}
	if conf.CNIVersion == "" {
		conf.CNIVersion = "1.0.0"
	}
	if conf.ManagedLabel == "" {
		conf.ManagedLabel = inject.ProxylessManagedLabel
	}
	if conf.ManagedLabelValue == "" {
		conf.ManagedLabelValue = inject.ProxylessManagedLabelValue
	}
	if conf.GRPCInboundPort == 0 {
		conf.GRPCInboundPort = inject.ProxylessGRPCInboundPort
	}
	if conf.IPTablesPath == "" {
		conf.IPTablesPath = defaultIPTablesPath
	}
	if conf.IPSetPath == "" {
		conf.IPSetPath = defaultIPSetPath
	}
	return conf, nil
}

func (c NetConf) KubeConfigPath() string {
	if c.Kubernetes.KubeConfig != "" {
		return c.Kubernetes.KubeConfig
	}
	return c.KubeConfig
}

func (c NetConf) StateDirectory() string {
	if c.StateDir != "" {
		return c.StateDir
	}
	return defaultStateDir
}

func ResultOutput(conf NetConf) []byte {
	if len(conf.PrevResult) > 0 {
		return append([]byte(nil), conf.PrevResult...)
	}
	result := map[string]string{"cniVersion": conf.CNIVersion}
	out, _ := json.Marshal(result)
	return out
}

func PodIPsFromPrevResult(prev json.RawMessage) []string {
	if len(prev) == 0 {
		return nil
	}
	var result struct {
		IPs []struct {
			Address string `json:"address"`
		} `json:"ips"`
	}
	if err := json.Unmarshal(prev, &result); err != nil {
		return nil
	}
	ips := make([]string, 0, len(result.IPs))
	for _, item := range result.IPs {
		ip := strings.TrimSpace(item.Address)
		if ip == "" {
			continue
		}
		if parsedIP, _, err := net.ParseCIDR(ip); err == nil {
			ips = append(ips, parsedIP.String())
			continue
		}
		if parsedIP := net.ParseIP(ip); parsedIP != nil {
			ips = append(ips, parsedIP.String())
		}
	}
	return ips
}

type Env struct {
	Command     string
	ContainerID string
	NetNS       string
	IfName      string
	Args        string
}

func EnvFromOS() Env {
	return Env{
		Command:     os.Getenv("CNI_COMMAND"),
		ContainerID: os.Getenv("CNI_CONTAINERID"),
		NetNS:       os.Getenv("CNI_NETNS"),
		IfName:      os.Getenv("CNI_IFNAME"),
		Args:        os.Getenv("CNI_ARGS"),
	}
}

func EnvHasKubernetesPod(env Env) bool {
	_, ok := PodRefFromCNIArgs(env.Args)
	return ok
}

type PodRef struct {
	Namespace string
	Name      string
}

func PodRefFromCNIArgs(args string) (PodRef, bool) {
	values := map[string]string{}
	for _, item := range strings.Split(args, ";") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		values[key] = value
	}
	ref := PodRef{
		Namespace: values["K8S_POD_NAMESPACE"],
		Name:      values["K8S_POD_NAME"],
	}
	return ref, ref.Namespace != "" && ref.Name != ""
}
