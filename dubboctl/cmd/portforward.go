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
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/apache/dubbo-kubernetes/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

// forwardToService port-forwards a local port to a running pod backing the
// given service and blocks until the context is canceled or an interrupt is
// received.
func forwardToService(ctx context.Context, client kube.CLIClient, out io.Writer, namespace, serviceName string, localPort, podPort int) error {
	pod, err := podForService(ctx, client, namespace, serviceName)
	if err != nil {
		return err
	}

	restConfig := client.RESTConfig()
	transport, upgrader, err := spdy.RoundTripperFor(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create SPDY round tripper: %v", err)
	}
	req := client.Kube().CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		SubResource("portforward")
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, req.URL())

	stopCh := make(chan struct{})
	readyCh := make(chan struct{})
	errCh := make(chan error, 1)

	fw, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", localPort, podPort)}, stopCh, readyCh, io.Discard, os.Stderr)
	if err != nil {
		return fmt.Errorf("failed to create port forwarder: %v", err)
	}
	go func() {
		errCh <- fw.ForwardPorts()
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("port-forward to %s/%s failed: %v", pod.Namespace, pod.Name, err)
	case <-readyCh:
	}
	fmt.Fprintf(out, "Forwarding http://localhost:%d -> pod %s/%s:%d (Ctrl+C to stop)\n", localPort, pod.Namespace, pod.Name, podPort)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case <-ctx.Done():
	case <-sigCh:
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("port-forward terminated: %v", err)
		}
	}
	close(stopCh)
	return nil
}

// podForService resolves a running pod backing the given service.
func podForService(ctx context.Context, client kube.CLIClient, namespace, serviceName string) (*corev1.Pod, error) {
	svc, err := client.Kube().CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("service %s/%s not found; is the addon installed? (dubboctl install --set profile=demo installs the observability stack): %v", namespace, serviceName, err)
	}
	if len(svc.Spec.Selector) == 0 {
		return nil, fmt.Errorf("service %s/%s has no selector", namespace, serviceName)
	}
	pods, err := client.Kube().CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(svc.Spec.Selector).String(),
		FieldSelector: "status.phase=Running",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods for service %s/%s: %v", namespace, serviceName, err)
	}
	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("no running pods found for service %s/%s", namespace, serviceName)
	}
	return &pods.Items[0], nil
}
