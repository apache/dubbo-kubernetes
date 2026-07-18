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
	"context"
	"encoding/json"
	"fmt"
)

type Plugin struct {
	PodInfoProvider PodInfoProvider
	RuleManager     RuleManager
	StateStore      StateStore
}

type PodInfo struct {
	Namespace string
	Name      string
	Labels    map[string]string
	IPs       []string
}

type PodInfoProvider interface {
	PodInfo(ctx context.Context, ref PodRef) (PodInfo, error)
}

type RuleManager interface {
	AddPodRules(ctx context.Context, podIP string) error
	DeletePodRules(ctx context.Context, podIP string) error
}

type StateStore interface {
	Write(PodState) error
	Read(containerID string) (PodState, error)
	Delete(containerID string) error
}

func (p Plugin) Run(ctx context.Context, env Env, conf NetConf) ([]byte, error) {
	switch env.Command {
	case "VERSION":
		return versionOutput(conf.CNIVersion), nil
	case "ADD", "CHECK":
		return p.addOrCheck(ctx, env, conf)
	case "DEL":
		return nil, p.del(ctx, env)
	case "":
		return nil, fmt.Errorf("CNI_COMMAND is required")
	default:
		return nil, fmt.Errorf("unsupported CNI_COMMAND %q", env.Command)
	}
}

func (p Plugin) addOrCheck(ctx context.Context, env Env, conf NetConf) ([]byte, error) {
	out := ResultOutput(conf)
	ref, ok := PodRefFromCNIArgs(env.Args)
	if !ok {
		return out, nil
	}
	if p.PodInfoProvider == nil {
		return out, nil
	}
	pod, err := p.PodInfoProvider.PodInfo(ctx, ref)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		// Ownership is not proven until the managed label is read.
		return out, nil
	}
	if !isManagedPod(conf, pod) {
		return out, nil
	}
	podIP := firstPodIP(conf, pod)
	if podIP == "" {
		return nil, fmt.Errorf("managed pod %s/%s has no IP in CNI result or Kubernetes status", ref.Namespace, ref.Name)
	}
	if p.RuleManager == nil {
		return nil, fmt.Errorf("rule manager is required")
	}
	if err := p.RuleManager.AddPodRules(ctx, podIP); err != nil {
		return nil, err
	}
	if p.StateStore != nil && env.ContainerID != "" {
		if err := p.StateStore.Write(PodState{
			ContainerID: env.ContainerID,
			Namespace:   ref.Namespace,
			Name:        ref.Name,
			IP:          podIP,
		}); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (p Plugin) del(ctx context.Context, env Env) error {
	if p.StateStore == nil || env.ContainerID == "" {
		return nil
	}
	state, err := p.StateStore.Read(env.ContainerID)
	if err != nil {
		if IsNotFound(err) {
			return nil
		}
		return err
	}
	if p.RuleManager != nil && state.IP != "" {
		if err := p.RuleManager.DeletePodRules(ctx, state.IP); err != nil {
			return err
		}
	}
	return p.StateStore.Delete(env.ContainerID)
}

func isManagedPod(conf NetConf, pod PodInfo) bool {
	if conf.ManagedLabel == "" {
		return false
	}
	return pod.Labels[conf.ManagedLabel] == conf.ManagedLabelValue
}

func firstPodIP(conf NetConf, pod PodInfo) string {
	if ips := PodIPsFromPrevResult(conf.PrevResult); len(ips) > 0 {
		return ips[0]
	}
	if len(pod.IPs) > 0 {
		return pod.IPs[0]
	}
	return ""
}

func versionOutput(cniVersion string) []byte {
	if cniVersion == "" {
		cniVersion = defaultCNIVersion
	}
	out, _ := json.Marshal(map[string]any{
		"cniVersion":        cniVersion,
		"supportedVersions": []string{"0.3.1", "0.4.0", "1.0.0", "1.1.0"},
	})
	return out
}
