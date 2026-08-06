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
	"os"
	"time"
)

type Plugin struct {
	PodInfoProvider PodInfoProvider
	RuleManager     RuleManager
	StateStore      StateStore
}

type PodInfo struct {
	Namespace     string
	Name          string
	Labels        map[string]string
	IPs           []string
	ExcludedPorts []int
}

type PodInfoProvider interface {
	PodInfo(ctx context.Context, ref PodRef) (PodInfo, error)
}

type RuleManager interface {
	AddPodRules(ctx context.Context, podIP string, excludedPorts []int) error
	DeletePodRules(ctx context.Context, podIP string, excludedPorts []int) error
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
		return p.unresolvedPod(conf, out, ref, fmt.Errorf("no Kubernetes client is configured"))
	}
	pod, err := p.lookupPod(ctx, conf, ref)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return p.unresolvedPod(conf, out, ref, err)
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
	if err := p.RuleManager.AddPodRules(ctx, podIP, pod.ExcludedPorts); err != nil {
		return nil, err
	}
	if p.StateStore != nil && env.ContainerID != "" {
		if err := p.StateStore.Write(PodState{
			ContainerID:   env.ContainerID,
			Namespace:     ref.Namespace,
			Name:          ref.Name,
			IP:            podIP,
			ExcludedPorts: pod.ExcludedPorts,
		}); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// lookupPod retries a failed lookup before giving up. Pod creation is a burst
// workload and the API server is the one dependency in this path, so a short
// retry converts most transient failures into a normal ADD.
func (p Plugin) lookupPod(ctx context.Context, conf NetConf, ref PodRef) (PodInfo, error) {
	attempts := conf.PodLookupAttempts()
	var err error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			timer := time.NewTimer(conf.PodLookupBackoff())
			select {
			case <-ctx.Done():
				timer.Stop()
				return PodInfo{}, ctx.Err()
			case <-timer.C:
			}
		}
		var pod PodInfo
		pod, err = p.PodInfoProvider.PodInfo(ctx, ref)
		if err == nil {
			return pod, nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return PodInfo{}, ctxErr
		}
	}
	return PodInfo{}, err
}

// unresolvedPod decides what to do when pod ownership could not be determined.
//
// Ownership is only knowable from the managed label, so an unreadable pod
// might not be a mesh pod at all. Failing the ADD would block it from
// starting, which puts non-mesh workloads at the mercy of a mesh component —
// the plugin is chained after the primary CNI precisely so that it never owns
// that decision. So ADD is allowed to proceed, loudly.
//
// The gap this leaves — a mesh pod that never got its rules, which ADD will
// not retry — is closed by the node agent's reconcile loop, which reads the
// managed pods on this node and fences any that are missing. Clusters that
// would rather stop scheduling than run with that window can set failClosed.
func (p Plugin) unresolvedPod(conf NetConf, out []byte, ref PodRef, cause error) ([]byte, error) {
	if conf.FailClosed {
		return nil, fmt.Errorf("determine whether pod %s/%s is mesh-managed: %w", ref.Namespace, ref.Name, cause)
	}
	fmt.Fprintf(os.Stderr,
		"dubbo-cni: allowing %s/%s without inbound rules because pod ownership could not be determined: %v; "+
			"the node agent reconcile loop will install them if the pod is mesh-managed\n",
		ref.Namespace, ref.Name, cause)
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
		if err := p.RuleManager.DeletePodRules(ctx, state.IP, state.ExcludedPorts); err != nil {
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
		cniVersion = "1.0.0"
	}
	out, _ := json.Marshal(map[string]any{
		"cniVersion":        cniVersion,
		"supportedVersions": []string{"0.3.1", "0.4.0", "1.0.0", "1.1.0"},
	})
	return out
}
