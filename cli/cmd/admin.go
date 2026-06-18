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

	"github.com/apache/dubbo-kubernetes/cli/pkg/cli"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type adminLogArgs struct {
	namespace  string
	deployment string
	container  string
	tail       int64
}

func AdminCmd(ctx cli.Context) *cobra.Command {
	command := &cobra.Command{
		Use:   "admin",
		Short: "Access dubbod admin information",
	}
	command.AddCommand(adminLogCmd(ctx))
	return command
}

func adminLogCmd(ctx cli.Context) *cobra.Command {
	args := &adminLogArgs{
		namespace:  ctx.Namespace(),
		deployment: "dubbod",
		container:  "execute",
		tail:       200,
	}

	command := &cobra.Command{
		Use:   "log",
		Short: "Print dubbod logs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := ctx.CLIClient()
			if err != nil {
				return err
			}
			return printDeploymentLogs(cmd.Context(), client.Kube(), *args, cmd.OutOrStdout())
		},
	}
	flags := command.Flags()
	flags.StringVarP(&args.namespace, "namespace", "n", args.namespace, "Namespace of the dubbod deployment")
	flags.StringVar(&args.deployment, "deployment", args.deployment, "Dubbod deployment name")
	flags.StringVar(&args.container, "container", args.container, "Preferred container name")
	flags.Int64Var(&args.tail, "tail", args.tail, "Number of recent log lines to print")
	return command
}

func printDeploymentLogs(ctx context.Context, client kubernetes.Interface, args adminLogArgs, writer io.Writer) error {
	pods, err := deploymentPods(ctx, client, args.namespace, args.deployment)
	if err != nil {
		return err
	}
	if len(pods) == 0 {
		return fmt.Errorf("no pods found for deployment %s/%s", args.namespace, args.deployment)
	}

	for _, pod := range pods {
		for _, container := range logContainers(pod, args.container) {
			if _, err := fmt.Fprintf(writer, "==> %s/%s:%s <==\n", pod.Namespace, pod.Name, container); err != nil {
				return err
			}
			raw, err := client.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
				Container:  container,
				TailLines:  &args.tail,
				Timestamps: true,
			}).DoRaw(ctx)
			if err != nil {
				return fmt.Errorf("get logs %s/%s container %s: %w", pod.Namespace, pod.Name, container, err)
			}
			if _, err := writer.Write(raw); err != nil {
				return err
			}
			if len(raw) == 0 || raw[len(raw)-1] != '\n' {
				if _, err := fmt.Fprintln(writer); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func deploymentPods(ctx context.Context, client kubernetes.Interface, namespace, name string) ([]corev1.Pod, error) {
	deployment, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get deployment %s/%s: %w", namespace, name, err)
	}
	if deployment.Spec.Selector == nil {
		return nil, fmt.Errorf("deployment %s/%s has no selector", namespace, name)
	}
	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return nil, fmt.Errorf("build pod selector for deployment %s/%s: %w", namespace, name, err)
	}
	if selector.Empty() {
		selector = klabels.SelectorFromSet(deployment.Spec.Template.Labels)
	}
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("list pods for deployment %s/%s: %w", namespace, name, err)
	}
	sort.SliceStable(pods.Items, func(i, j int) bool {
		return pods.Items[i].Name < pods.Items[j].Name
	})
	return pods.Items, nil
}

func logContainers(pod corev1.Pod, preferred string) []string {
	if preferred != "" {
		for _, container := range pod.Spec.Containers {
			if container.Name == preferred {
				return []string{preferred}
			}
		}
	}
	out := make([]string, 0, len(pod.Spec.Containers))
	for _, container := range pod.Spec.Containers {
		out = append(out, container.Name)
	}
	return out
}
