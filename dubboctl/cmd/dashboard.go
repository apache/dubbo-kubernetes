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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

const dashboardNamespace = "dubbo-system"

type dashboardArgs struct {
	manifests string
	wait      time.Duration
}

type appliedObject struct {
	Kind      string
	Name      string
	Namespace string
}

func DashboardCmd(ctx cli.Context) *cobra.Command {
	args := &dashboardArgs{
		manifests: "samples/addons",
		wait:      2 * time.Minute,
	}

	command := &cobra.Command{
		Use:   "dashboard",
		Short: "Start the sample Prometheus and Grafana dashboard",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := ctx.CLIClient()
			if err != nil {
				return err
			}

			files, err := dashboardManifestFiles(args.manifests)
			if err != nil {
				return err
			}

			var applied []appliedObject
			for _, file := range files {
				objects, err := applyManifestFile(cmd.Context(), client, file)
				if err != nil {
					return err
				}
				applied = append(applied, objects...)
			}
			if err := printAppliedDashboard(cmd.OutOrStdout(), applied); err != nil {
				return err
			}

			if args.wait > 0 {
				if err := waitForDashboard(cmd.Context(), client.Kube(), dashboardNamespace, args.wait); err != nil {
					return err
				}
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), "dashboard ready"); err != nil {
					return err
				}
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "prometheus: kubectl -n %s port-forward svc/prometheus 9090:9090\ngrafana: kubectl -n %s port-forward svc/grafana 3000:3000\n", dashboardNamespace, dashboardNamespace)
			return err
		},
	}
	command.Flags().StringVar(&args.manifests, "manifests", args.manifests, "Dashboard manifest file or samples/addons directory")
	command.Flags().DurationVar(&args.wait, "wait", args.wait, "Maximum time to wait for Prometheus and Grafana deployments; 0 disables waiting")
	return command
}

func dashboardManifestFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{path}, nil
	}

	prometheus := filepath.Join(path, "prometheus.yaml")
	if _, err := os.Stat(prometheus); err != nil {
		prometheus = filepath.Join(path, "metrics.yaml")
		if _, err := os.Stat(prometheus); err != nil {
			return nil, fmt.Errorf("dashboard prometheus manifest not found in %s", path)
		}
	}

	grafana := filepath.Join(path, "grafana.yaml")
	if _, err := os.Stat(grafana); err != nil {
		return nil, fmt.Errorf("dashboard grafana manifest not found in %s", path)
	}
	return []string{prometheus, grafana}, nil
}

func applyManifestFile(ctx context.Context, client kube.CLIClient, file string) ([]appliedObject, error) {
	objects, err := readManifestObjects(file)
	if err != nil {
		return nil, err
	}
	applied := make([]appliedObject, 0, len(objects))
	for _, object := range objects {
		item, err := applyDashboardObject(ctx, client, object)
		if err != nil {
			return nil, err
		}
		applied = append(applied, item)
	}
	return applied, nil
}

func readManifestObjects(path string) ([]*unstructured.Unstructured, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := utilyaml.NewYAMLReader(bufio.NewReader(file))
	out := []*unstructured.Unstructured{}
	for {
		doc, err := reader.Read()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read manifest %s: %w", path, err)
		}
		doc = bytes.TrimSpace(doc)
		if len(doc) == 0 {
			continue
		}
		jsonDoc, err := yaml.YAMLToJSON(doc)
		if err != nil {
			return nil, fmt.Errorf("decode manifest %s: %w", path, err)
		}
		object := &unstructured.Unstructured{}
		if err := json.Unmarshal(jsonDoc, object); err != nil {
			return nil, fmt.Errorf("decode manifest %s: %w", path, err)
		}
		if object.GetKind() == "" || object.GetAPIVersion() == "" || object.GetName() == "" {
			return nil, fmt.Errorf("manifest %s contains an object without apiVersion, kind, or metadata.name", path)
		}
		out = append(out, object)
	}
}

func applyDashboardObject(ctx context.Context, client kube.CLIClient, object *unstructured.Unstructured) (appliedObject, error) {
	data, err := json.Marshal(object.Object)
	if err != nil {
		return appliedObject{}, err
	}
	resource, err := client.DynamicClientFor(object.GroupVersionKind(), object, object.GetNamespace())
	if err != nil {
		return appliedObject{}, err
	}
	force := true
	applied, err := resource.Patch(ctx, object.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
		FieldManager: "dubboctl-dashboard",
		Force:        &force,
	})
	if err != nil {
		return appliedObject{}, fmt.Errorf("apply %s %s/%s: %w", object.GetKind(), object.GetNamespace(), object.GetName(), err)
	}
	kind := applied.GetKind()
	if kind == "" {
		kind = object.GetKind()
	}
	namespace := applied.GetNamespace()
	if namespace == "" {
		namespace = object.GetNamespace()
	}
	return appliedObject{
		Kind:      kind,
		Name:      applied.GetName(),
		Namespace: namespace,
	}, nil
}

func printAppliedDashboard(writer io.Writer, objects []appliedObject) error {
	for _, object := range objects {
		name := object.Name
		if object.Namespace != "" {
			name = object.Namespace + "/" + name
		}
		if _, err := fmt.Fprintf(writer, "applied %s %s\n", object.Kind, name); err != nil {
			return err
		}
	}
	return nil
}

func waitForDashboard(ctx context.Context, client kubernetes.Interface, namespace string, timeout time.Duration) error {
	for _, name := range []string{"prometheus", "grafana"} {
		if err := waitForAvailableDeployment(ctx, client, namespace, name, timeout); err != nil {
			return err
		}
	}
	return nil
}

func waitForAvailableDeployment(ctx context.Context, client kubernetes.Interface, namespace, name string, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(ctx, 2*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		deployment, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return deploymentAvailable(deployment), nil
	})
}

func deploymentAvailable(deployment *appsv1.Deployment) bool {
	desired := int32(1)
	if deployment.Spec.Replicas != nil {
		desired = *deployment.Spec.Replicas
	}
	return deployment.Status.AvailableReplicas >= desired
}
