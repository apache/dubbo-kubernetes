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

package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	cniconstants "github.com/apache/dubbo-kubernetes/cni/pkg/constants"
)

const (
	CmdAdd   = "ADD"
	CmdDel   = "DEL"
	CmdCheck = "CHECK"
)

// PluginConf mirrors the JSON configuration passed to a chained CNI plugin on stdin.
type PluginConf struct {
	CNIVersion    string          `json:"cniVersion"`
	Name          string          `json:"name"`
	Type          string          `json:"type"`
	PrevResult    json.RawMessage `json:"prevResult,omitempty"`
	SocketPath    string          `json:"socketPath,omitempty"`
	LogLevel      string          `json:"logLevel,omitempty"`
	FailOpen      bool            `json:"failOpen,omitempty"`
	ExcludeSpaces []string        `json:"excludeNamespaces,omitempty"`
}

// PodEvent is sent by dubbo-cni over the local Unix socket to dubbo-cni-node.
type PodEvent struct {
	Command     string `json:"command"`
	ContainerID string `json:"containerId"`
	NetNS       string `json:"netns"`
	IfName      string `json:"ifName"`
	PodName     string `json:"podName"`
	Namespace   string `json:"namespace"`
	PodUID      string `json:"podUid,omitempty"`
	PodIP       string `json:"podIp,omitempty"`
}

// Env captures the CNI environment variables defined by the CNI specification.
type Env struct {
	Command     string
	ContainerID string
	NetNS       string
	IfName      string
	Args        string
}

// EnvFromOS reads the standard CNI environment variables.
func EnvFromOS() Env {
	return Env{
		Command:     os.Getenv("CNI_COMMAND"),
		ContainerID: os.Getenv("CNI_CONTAINERID"),
		NetNS:       os.Getenv("CNI_NETNS"),
		IfName:      os.Getenv("CNI_IFNAME"),
		Args:        os.Getenv("CNI_ARGS"),
	}
}

// ParseK8sArgs parses semicolon-delimited CNI_ARGS (e.g. K8S_POD_NAME=foo;K8S_POD_NAMESPACE=bar).
func ParseK8sArgs(raw string) map[string]string {
	out := map[string]string{}
	for _, pair := range strings.Split(raw, ";") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		k, v, ok := strings.Cut(pair, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return out
}

// ExtractPodIP extracts the first IPv4 (or first IP) address from a CNI prevResult JSON blob.
func ExtractPodIP(prevResult json.RawMessage) string {
	if len(prevResult) == 0 {
		return ""
	}
	var parsed struct {
		IPs []struct {
			Address string `json:"address"`
		} `json:"ips"`
	}
	if err := json.Unmarshal(prevResult, &parsed); err != nil {
		return ""
	}
	first := ""
	for _, item := range parsed.IPs {
		addr := item.Address
		if ip, _, err := net.ParseCIDR(addr); err == nil {
			addr = ip.String()
		}
		if addr == "" {
			continue
		}
		if first == "" {
			first = addr
		}
		if parsedIP := net.ParseIP(addr); parsedIP != nil && parsedIP.To4() != nil {
			return parsedIP.String()
		}
	}
	return first
}

// EventSender sends CNI ADD/DEL events to dubbo-cni-node.
type EventSender interface {
	Send(ctx context.Context, socketPath string, event PodEvent) error
}

// UDSEventSender delivers PodEvents over HTTP-over-Unix-domain-socket.
type UDSEventSender struct {
	Timeout time.Duration
}

// Send posts a PodEvent to the node agent's Unix domain socket.
func (s *UDSEventSender) Send(ctx context.Context, socketPath string, event PodEvent) error {
	if socketPath == "" {
		socketPath = cniconstants.DefaultSocketPath
	}
	timeout := s.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: func(dialCtx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(dialCtx, "unix", socketPath)
			},
		},
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal CNI pod event: %w", err)
	}
	endpoint := "http://unix/v1/pods/add"
	if event.Command == CmdDel {
		endpoint = "http://unix/v1/pods/del"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("notify dubbo-cni-node at %s: %w", socketPath, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("dubbo-cni-node rejected %s (%d): %s", event.Command, resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	return nil
}

// Execute runs a single CNI invocation, reading stdin config and writing the CNI result to stdout.
func Execute(ctx context.Context, env Env, stdin io.Reader, stdout io.Writer, sender EventSender) error {
	raw, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("read CNI stdin: %w", err)
	}
	var conf PluginConf
	if err := json.Unmarshal(raw, &conf); err != nil {
		return fmt.Errorf("parse CNI config: %w", err)
	}
	if conf.CNIVersion == "" {
		conf.CNIVersion = "1.0.0"
	}
	if conf.SocketPath == "" {
		conf.SocketPath = cniconstants.DefaultSocketPath
	}
	if sender == nil {
		sender = &UDSEventSender{}
	}

	k8sArgs := ParseK8sArgs(env.Args)
	podName := k8sArgs["K8S_POD_NAME"]
	namespace := k8sArgs["K8S_POD_NAMESPACE"]
	podUID := k8sArgs["K8S_POD_UID"]

	switch env.Command {
	case CmdAdd:
		if podName != "" && namespace != "" && !isExcludedNamespace(namespace, conf.ExcludeSpaces) {
			event := PodEvent{
				Command:     CmdAdd,
				ContainerID: env.ContainerID,
				NetNS:       env.NetNS,
				IfName:      env.IfName,
				PodName:     podName,
				Namespace:   namespace,
				PodUID:      podUID,
				PodIP:       ExtractPodIP(conf.PrevResult),
			}
			if err := sender.Send(ctx, conf.SocketPath, event); err != nil && !conf.FailOpen {
				return err
			}
		}
		return writeResult(stdout, conf)
	case CmdDel:
		if podName != "" && namespace != "" {
			event := PodEvent{
				Command:     CmdDel,
				ContainerID: env.ContainerID,
				NetNS:       env.NetNS,
				IfName:      env.IfName,
				PodName:     podName,
				Namespace:   namespace,
				PodUID:      podUID,
			}
			// DEL must be best-effort per CNI spec so pod teardown never blocks.
			_ = sender.Send(ctx, conf.SocketPath, event)
		}
		return nil
	case CmdCheck:
		return nil
	default:
		return fmt.Errorf("unsupported CNI_COMMAND %q", env.Command)
	}
}

func isExcludedNamespace(ns string, excluded []string) bool {
	if ns == "kube-system" || ns == "dubbo-system" {
		return true
	}
	for _, ex := range excluded {
		if strings.TrimSpace(ex) == ns {
			return true
		}
	}
	return false
}

func writeResult(stdout io.Writer, conf PluginConf) error {
	if len(conf.PrevResult) > 0 && string(bytes.TrimSpace(conf.PrevResult)) != "null" {
		_, err := stdout.Write(conf.PrevResult)
		return err
	}
	result := map[string]any{
		"cniVersion": conf.CNIVersion,
	}
	return json.NewEncoder(stdout).Encode(result)
}

// CNIError formats a CNI-spec compliant JSON error response.
type CNIError struct {
	CNIVersion string `json:"cniVersion"`
	Code       uint   `json:"code"`
	Msg        string `json:"msg"`
	Details    string `json:"details,omitempty"`
}

func WriteError(w io.Writer, err error) {
	payload := CNIError{
		CNIVersion: "1.0.0",
		Code:       100,
		Msg:        err.Error(),
	}
	_ = json.NewEncoder(w).Encode(payload)
}
