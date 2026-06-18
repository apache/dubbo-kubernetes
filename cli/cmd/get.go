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
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/apache/dubbo-kubernetes/cli/pkg/cli"
	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
	"github.com/kdubbo/api/annotation"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/client-go/kubernetes"
)

type getPodsArgs struct {
	namespace string
}

func GetCmd(ctx cli.Context) *cobra.Command {
	command := &cobra.Command{
		Use:   "get",
		Short: "Display Dubbo resources",
	}
	command.AddCommand(getPodsCmd(ctx))
	return command
}

func getPodsCmd(ctx cli.Context) *cobra.Command {
	args := &getPodsArgs{}
	command := &cobra.Command{
		Use:     "pods",
		Aliases: []string{"pod", "po"},
		Short:   "List pods injected by Dubbo",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := ctx.CLIClient()
			if err != nil {
				return err
			}
			pods, err := listInjectedPods(cmd.Context(), client.Kube(), args.namespace)
			if err != nil {
				return err
			}
			return printInjectedPods(cmd.OutOrStdout(), pods, time.Now())
		},
	}
	command.Flags().StringVarP(&args.namespace, "namespace", "n", "", "Namespace to scan; empty scans all namespaces")
	return command
}

func listInjectedPods(ctx context.Context, client kubernetes.Interface, namespace string) ([]corev1.Pod, error) {
	listNamespace := namespace
	if listNamespace == "" {
		listNamespace = metav1.NamespaceAll
	}
	pods, err := client.CoreV1().Pods(listNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	out := make([]corev1.Pod, 0, len(pods.Items))
	for _, pod := range pods.Items {
		if isInjectedPod(pod) {
			out = append(out, pod)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Namespace == out[j].Namespace {
			return out[i].Name < out[j].Name
		}
		return out[i].Namespace < out[j].Namespace
	})
	return out, nil
}

func isInjectedPod(pod corev1.Pod) bool {
	if pod.Annotations[annotation.OrgApacheDubboProxylessStatus.Name] != "" {
		return true
	}
	if containsCSV(pod.Annotations[annotation.OrgApacheDubboInjectTemplates.Name], inject.ProxylessGRPCTemplateName) {
		return true
	}
	if hasVolume(pod, inject.ProxylessXDSVolumeName) {
		return true
	}
	for _, container := range allContainers(pod) {
		if container.Name == inject.ProxylessXServerContainerName {
			return true
		}
		for _, env := range container.Env {
			switch env.Name {
			case "GRPC_XDS_BOOTSTRAP", inject.ProxylessGRPCConfigEnvName, inject.ProxylessXDSAddressEnvName:
				return true
			}
		}
	}
	return false
}

func printInjectedPods(writer io.Writer, pods []corev1.Pod, now time.Time) error {
	table := tabwriter.NewWriter(writer, 0, 8, 2, ' ', 0)
	if _, err := fmt.Fprintln(table, "NAMESPACE\tNAME\tREADY\tSTATUS\tRESTARTS\tAGE\tIP\tNODE\tTEMPLATES"); err != nil {
		return err
	}
	for _, pod := range pods {
		if _, err := fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%d\t%s\t%s\t%s\t%s\n",
			pod.Namespace,
			pod.Name,
			readyContainers(pod),
			podDisplayStatus(pod),
			restartCount(pod),
			duration.HumanDuration(now.Sub(pod.CreationTimestamp.Time)),
			pod.Status.PodIP,
			pod.Spec.NodeName,
			podTemplates(pod),
		); err != nil {
			return err
		}
	}
	return table.Flush()
}

func allContainers(pod corev1.Pod) []corev1.Container {
	out := make([]corev1.Container, 0, len(pod.Spec.InitContainers)+len(pod.Spec.Containers))
	out = append(out, pod.Spec.InitContainers...)
	out = append(out, pod.Spec.Containers...)
	return out
}

func hasVolume(pod corev1.Pod, name string) bool {
	for _, volume := range pod.Spec.Volumes {
		if volume.Name == name {
			return true
		}
	}
	return false
}

func containsCSV(raw, want string) bool {
	for _, item := range strings.Split(raw, ",") {
		if strings.TrimSpace(item) == want {
			return true
		}
	}
	return false
}

func readyContainers(pod corev1.Pod) string {
	ready := 0
	for _, status := range pod.Status.ContainerStatuses {
		if status.Ready {
			ready++
		}
	}
	return fmt.Sprintf("%d/%d", ready, len(pod.Spec.Containers))
}

func restartCount(pod corev1.Pod) int32 {
	var count int32
	for _, status := range pod.Status.InitContainerStatuses {
		count += status.RestartCount
	}
	for _, status := range pod.Status.ContainerStatuses {
		count += status.RestartCount
	}
	return count
}

func podDisplayStatus(pod corev1.Pod) string {
	if pod.DeletionTimestamp != nil {
		return "Terminating"
	}
	for _, status := range pod.Status.ContainerStatuses {
		if status.State.Waiting != nil && status.State.Waiting.Reason != "" {
			return status.State.Waiting.Reason
		}
		if status.State.Terminated != nil && status.State.Terminated.Reason != "" {
			return status.State.Terminated.Reason
		}
	}
	if pod.Status.Phase == "" {
		return "Unknown"
	}
	return string(pod.Status.Phase)
}

func podTemplates(pod corev1.Pod) string {
	if templates := strings.TrimSpace(pod.Annotations[annotation.OrgApacheDubboInjectTemplates.Name]); templates != "" {
		return templates
	}
	if isInjectedPod(pod) {
		return inject.ProxylessGRPCTemplateName
	}
	return ""
}
