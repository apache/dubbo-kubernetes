//
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

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/apache/dubbo-kubernetes/cli/pkg/cli"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const dubbodMonitoringPort = "8080"

// proxySyncStatus mirrors the JSON emitted by dubbod's /debug/syncz endpoint.
type proxySyncStatus struct {
	ProxyID      string `json:"proxy"`
	ClusterID    string `json:"cluster_id,omitempty"`
	WatchedTypes []struct {
		TypeURL      string `json:"type_url"`
		NonceSent    string `json:"nonce_sent,omitempty"`
		NonceAcked   string `json:"nonce_acked,omitempty"`
		Synced       bool   `json:"synced"`
		ResourceSize int    `json:"resources"`
		LastError    string `json:"last_error,omitempty"`
	} `json:"watched_types"`
}

// ProxyStatusCmd reports the xDS synchronization status of every proxy
// connected to dubbod, based on the /debug/syncz endpoint.
func ProxyStatusCmd(ctx cli.Context) *cobra.Command {
	command := &cobra.Command{
		Use:     "proxy-status",
		Aliases: []string{"ps"},
		Short:   "Retrieves the xDS sync status of each proxy in the mesh",
		Long: `Retrieves the last sent and last acknowledged xDS resource versions for each proxy
connected to dubbod. A resource type is SYNCED when the proxy has acknowledged
the most recent configuration pushed by dubbod.`,
		Example: `  # Show the sync status of all proxies
  dubboctl proxy-status`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := ctx.CLIClient()
			if err != nil {
				return fmt.Errorf("failed to create Kubernetes client: %v", err)
			}
			statuses, err := fetchSyncStatuses(cmd.Context(), client, ctx.Namespace())
			if err != nil {
				return err
			}
			writeProxyStatusTable(cmd.OutOrStdout(), statuses)
			return nil
		},
	}
	return command
}

// fetchSyncStatuses queries /debug/syncz on every dubbod pod through the
// Kubernetes API server proxy and merges the results.
func fetchSyncStatuses(ctx context.Context, client kube.CLIClient, namespace string) ([]proxySyncStatus, error) {
	pods, err := client.Kube().CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=dubbod",
		FieldSelector: "status.phase=Running",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list dubbod pods in namespace %q: %v", namespace, err)
	}
	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("no running dubbod pods found in namespace %q; is the control plane installed?", namespace)
	}
	seen := map[string]bool{}
	all := []proxySyncStatus{}
	var errs []string
	for _, pod := range pods.Items {
		data, err := proxyGetDebugEndpoint(ctx, client, pod, "/debug/syncz")
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", pod.Name, err))
			continue
		}
		statuses, err := parseSyncStatuses(data)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", pod.Name, err))
			continue
		}
		for _, s := range statuses {
			if seen[s.ProxyID] {
				continue
			}
			seen[s.ProxyID] = true
			all = append(all, s)
		}
	}
	if len(all) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("failed to fetch sync status from dubbod: %s", strings.Join(errs, "; "))
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ProxyID < all[j].ProxyID })
	return all, nil
}

func proxyGetDebugEndpoint(ctx context.Context, client kube.CLIClient, pod corev1.Pod, path string) ([]byte, error) {
	return client.Kube().CoreV1().Pods(pod.Namespace).
		ProxyGet("http", pod.Name, dubbodMonitoringPort, path, nil).
		DoRaw(ctx)
}

func writeProxyStatusTable(out io.Writer, statuses []proxySyncStatus) {
	if len(statuses) == 0 {
		fmt.Fprintln(out, "No proxies are connected to dubbod.")
		return
	}
	w := tabwriter.NewWriter(out, 0, 8, 4, ' ', 0)
	fmt.Fprintln(w, "NAME\tCLUSTER\tCDS\tLDS\tEDS\tRDS")
	for _, s := range statuses {
		byType := map[string]string{"CDS": "NOT WATCHED", "LDS": "NOT WATCHED", "EDS": "NOT WATCHED", "RDS": "NOT WATCHED"}
		for _, wt := range s.WatchedTypes {
			short := shortTypeName(wt.TypeURL)
			if _, ok := byType[short]; !ok {
				continue
			}
			switch {
			case wt.LastError != "":
				byType[short] = "ERROR"
			case wt.Synced:
				byType[short] = "SYNCED"
			case wt.NonceSent == "":
				byType[short] = "NOT SENT"
			default:
				byType[short] = "STALE"
			}
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", s.ProxyID, s.ClusterID, byType["CDS"], byType["LDS"], byType["EDS"], byType["RDS"])
	}
	w.Flush()
}

func shortTypeName(typeURL string) string {
	switch {
	case strings.Contains(typeURL, "ClusterLoadAssignment"), strings.Contains(typeURL, "Endpoint"):
		return "EDS"
	case strings.Contains(typeURL, "Cluster"):
		return "CDS"
	case strings.Contains(typeURL, "Listener"):
		return "LDS"
	case strings.Contains(typeURL, "Route"):
		return "RDS"
	default:
		return typeURL
	}
}

func parseSyncStatuses(data []byte) ([]proxySyncStatus, error) {
	statuses := []proxySyncStatus{}
	if err := json.Unmarshal(data, &statuses); err != nil {
		return nil, fmt.Errorf("failed to parse dubbod sync status: %v", err)
	}
	return statuses, nil
}
