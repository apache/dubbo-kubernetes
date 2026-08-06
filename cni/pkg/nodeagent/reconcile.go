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
	"fmt"
	"os"
	"time"
)

const defaultReconcileInterval = 60 * time.Second

// StateLister enumerates the pods this node has installed rules for.
type StateLister interface {
	List() ([]PodState, error)
}

// FenceReconciler restores the inbound fence from persisted pod state.
type FenceReconciler interface {
	Reconcile(ctx context.Context, states []PodState) error
}

// DefaultReconcileInterval is the period used when none is configured.
func DefaultReconcileInterval() time.Duration {
	return defaultReconcileInterval
}

// ClusterPodLister reports the mesh-managed pods currently on this node.
type ClusterPodLister interface {
	ManagedPodsOnNode(ctx context.Context, nodeName, label, value string) ([]PodState, error)
}

// ClusterSource binds a lister to the node and label it should query.
type ClusterSource struct {
	Lister     ClusterPodLister
	NodeName   string
	Label      string
	LabelValue string
}

// ReconcileOnce rebuilds the fence from the local state store, plus the
// cluster's view of this node when one is available.
//
// The two sources answer different questions. The local store records what CNI
// ADD actually installed. The cluster records what should be fenced, including
// pods whose ADD could not read them and therefore installed nothing. Merging
// them is what turns a permissive ADD into a bounded gap rather than a
// permanent hole.
func ReconcileOnce(ctx context.Context, lister StateLister, reconciler FenceReconciler, cluster *ClusterSource) error {
	if lister == nil || reconciler == nil {
		return fmt.Errorf("state lister and reconciler are required")
	}
	states, err := lister.List()
	if err != nil {
		return fmt.Errorf("list pod state: %w", err)
	}
	if cluster != nil && cluster.Lister != nil {
		clusterStates, err := cluster.Lister.ManagedPodsOnNode(ctx, cluster.NodeName, cluster.Label, cluster.LabelValue)
		if err != nil {
			// The local store still describes every pod ADD handled, so
			// reconciling from it alone is better than skipping this round.
			fmt.Fprintf(os.Stderr, "dubbo-cni: listing managed pods failed, reconciling from local state only: %v\n", err)
		} else {
			states = mergePodStates(states, clusterStates)
		}
	}
	return reconciler.Reconcile(ctx, states)
}

// mergePodStates unions both sources by pod IP. The cluster view wins on
// conflict because it reflects the pod's current annotations.
func mergePodStates(local, cluster []PodState) []PodState {
	byIP := make(map[string]PodState, len(local)+len(cluster))
	order := make([]string, 0, len(local)+len(cluster))
	for _, states := range [][]PodState{local, cluster} {
		for _, state := range states {
			if state.IP == "" {
				continue
			}
			if _, seen := byIP[state.IP]; !seen {
				order = append(order, state.IP)
			}
			byIP[state.IP] = state
		}
	}
	merged := make([]PodState, 0, len(order))
	for _, ip := range order {
		merged = append(merged, byIP[ip])
	}
	return merged
}

// ReconcileLoop keeps the fence in place for pods that were already running.
//
// The CNI ADD hook fires once per pod sandbox, but ipset and iptables state is
// lost on a node restart and can be flushed by unrelated tooling. Nothing
// replays ADD for running pods, so without this loop the fence stays missing
// until every pod happens to be recreated.
func ReconcileLoop(ctx context.Context, lister StateLister, reconciler FenceReconciler, cluster *ClusterSource, interval time.Duration) error {
	if interval <= 0 {
		interval = defaultReconcileInterval
	}
	if err := ReconcileOnce(ctx, lister, reconciler, cluster); err != nil {
		// A failure here must not stop the loop: the node may simply have no
		// state yet, and the next tick can still succeed.
		fmt.Fprintf(os.Stderr, "dubbo-cni: initial fence reconcile failed: %v\n", err)
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := ReconcileOnce(ctx, lister, reconciler, cluster); err != nil {
				fmt.Fprintf(os.Stderr, "dubbo-cni: fence reconcile failed: %v\n", err)
			}
		}
	}
}
