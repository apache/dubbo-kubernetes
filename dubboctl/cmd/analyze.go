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
	"strings"

	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/cli"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type analyzeLevel string

const (
	levelError   analyzeLevel = "Error"
	levelWarning analyzeLevel = "Warning"
	levelInfo    analyzeLevel = "Info"
)

type analyzeMessage struct {
	Level    analyzeLevel
	Resource string
	Message  string
}

// AnalyzeCmd inspects live mesh configuration and reports likely
// misconfigurations that pass schema validation but will not behave as intended.
func AnalyzeCmd(ctx cli.Context) *cobra.Command {
	var namespace string
	var allNamespaces bool
	command := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze Dubbo mesh configuration and report potential issues",
		Long: `Analyze inspects HTTPRoutes, security policies and circuit breaker policies in the
cluster and reports references to missing services, selectors that match no
workloads, and authentication configurations that are not actually enforced.`,
		Example: `  # Analyze the default namespace
  dubboctl analyze

  # Analyze a specific namespace
  dubboctl analyze -n backend

  # Analyze all namespaces
  dubboctl analyze -A`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := ctx.CLIClient()
			if err != nil {
				return fmt.Errorf("failed to create Kubernetes client: %v", err)
			}
			ns := namespace
			if allNamespaces {
				ns = metav1.NamespaceAll
			}
			msgs, err := runAnalyzers(cmd.Context(), client, ns)
			if err != nil {
				return err
			}
			writeAnalyzeMessages(cmd.OutOrStdout(), msgs)
			for _, m := range msgs {
				if m.Level == levelError {
					return fmt.Errorf("analysis reported errors")
				}
			}
			return nil
		},
	}
	command.PersistentFlags().StringVarP(&namespace, "namespace", "n", "default", "Namespace to analyze")
	command.PersistentFlags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "Analyze all namespaces")
	return command
}

func runAnalyzers(ctx context.Context, client kube.CLIClient, namespace string) ([]analyzeMessage, error) {
	msgs := []analyzeMessage{}

	services, err := client.Kube().CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %v", err)
	}
	pods, err := client.Kube().CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %v", err)
	}

	msgs = append(msgs, analyzeHTTPRoutes(ctx, client, namespace, services.Items)...)
	msgs = append(msgs, analyzeSecurityPolicies(ctx, client, namespace, pods.Items)...)
	msgs = append(msgs, analyzeCircuitBreakerPolicies(ctx, client, namespace, services.Items)...)
	return msgs, nil
}

// analyzeHTTPRoutes reports backendRefs pointing at services or ports that do not exist.
func analyzeHTTPRoutes(ctx context.Context, client kube.CLIClient, namespace string, services []corev1.Service) []analyzeMessage {
	msgs := []analyzeMessage{}
	routes, err := client.GatewayAPI().GatewayV1().HTTPRoutes(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return []analyzeMessage{{levelWarning, "HTTPRoute", fmt.Sprintf("failed to list HTTPRoutes: %v", err)}}
	}
	svcIndex := map[string]corev1.Service{}
	for _, svc := range services {
		svcIndex[svc.Namespace+"/"+svc.Name] = svc
	}
	for _, route := range routes.Items {
		resource := fmt.Sprintf("HTTPRoute %s/%s", route.Namespace, route.Name)
		for ruleIdx, rule := range route.Spec.Rules {
			for _, ref := range rule.BackendRefs {
				if ref.Kind != nil && *ref.Kind != "Service" {
					continue
				}
				refNs := route.Namespace
				if ref.Namespace != nil {
					refNs = string(*ref.Namespace)
				}
				key := refNs + "/" + string(ref.Name)
				svc, found := svcIndex[key]
				if !found {
					// The service may live in an unanalyzed namespace; only flag same-namespace misses as errors.
					if refNs == route.Namespace || namespace == metav1.NamespaceAll {
						msgs = append(msgs, analyzeMessage{levelError, resource,
							fmt.Sprintf("rule[%d] references service %q which does not exist", ruleIdx, key)})
					}
					continue
				}
				if ref.Port != nil {
					portFound := false
					for _, p := range svc.Spec.Ports {
						if p.Port == *ref.Port {
							portFound = true
							break
						}
					}
					if !portFound {
						msgs = append(msgs, analyzeMessage{levelError, resource,
							fmt.Sprintf("rule[%d] references port %d which is not defined on service %q", ruleIdx, *ref.Port, key)})
					}
				}
			}
		}
	}
	return msgs
}

// analyzeSecurityPolicies reports selectors matching no pods and JWT rules that
// are validated but not enforced by any authorization policy.
func analyzeSecurityPolicies(ctx context.Context, client kube.CLIClient, namespace string, pods []corev1.Pod) []analyzeMessage {
	msgs := []analyzeMessage{}
	security := client.Dubbo().SecurityV1alpha3()

	matchesAnyPod := func(policyNs string, matchLabels map[string]string) bool {
		selector := labels.SelectorFromSet(matchLabels)
		for _, pod := range pods {
			if policyNs != metav1.NamespaceAll && pod.Namespace != policyNs {
				continue
			}
			if selector.Matches(labels.Set(pod.Labels)) {
				return true
			}
		}
		return false
	}

	authzPolicies, err := security.AuthorizationPolicies(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		msgs = append(msgs, analyzeMessage{levelWarning, "AuthorizationPolicy", fmt.Sprintf("failed to list: %v", err)})
	} else {
		for _, policy := range authzPolicies.Items {
			resource := fmt.Sprintf("AuthorizationPolicy %s/%s", policy.Namespace, policy.Name)
			if sel := policy.Spec.GetSelector(); sel != nil && len(sel.GetMatchLabels()) > 0 {
				if !matchesAnyPod(policy.Namespace, sel.GetMatchLabels()) {
					msgs = append(msgs, analyzeMessage{levelWarning, resource,
						fmt.Sprintf("selector %v matches no pods; the policy has no effect", sel.GetMatchLabels())})
				}
			}
		}
	}

	reqAuths, err := security.RequestAuthentications(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		msgs = append(msgs, analyzeMessage{levelWarning, "RequestAuthentication", fmt.Sprintf("failed to list: %v", err)})
	} else {
		for _, ra := range reqAuths.Items {
			resource := fmt.Sprintf("RequestAuthentication %s/%s", ra.Namespace, ra.Name)
			if sel := ra.Spec.GetSelector(); sel != nil && len(sel.GetMatchLabels()) > 0 {
				if !matchesAnyPod(ra.Namespace, sel.GetMatchLabels()) {
					msgs = append(msgs, analyzeMessage{levelWarning, resource,
						fmt.Sprintf("selector %v matches no pods; the policy has no effect", sel.GetMatchLabels())})
				}
			}
			if len(ra.Spec.GetJwtRules()) == 0 {
				continue
			}
			// JWT tokens are validated when present, but requests without tokens
			// pass unless an ALLOW AuthorizationPolicy requires request principals.
			enforced := false
			if authzPolicies != nil {
				for _, policy := range authzPolicies.Items {
					if policy.Namespace != ra.Namespace {
						continue
					}
					for _, rule := range policy.Spec.GetRules() {
						for _, from := range rule.GetFrom() {
							if from.GetSource() != nil && len(from.GetSource().GetRequestPrincipals()) > 0 {
								enforced = true
							}
						}
					}
				}
			}
			if !enforced {
				msgs = append(msgs, analyzeMessage{levelWarning, resource,
					"JWT tokens are validated but not required: no AuthorizationPolicy in this namespace restricts requestPrincipals, so requests without a token are still allowed"})
			}
		}
	}

	peerAuths, err := security.PeerAuthentications(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		msgs = append(msgs, analyzeMessage{levelWarning, "PeerAuthentication", fmt.Sprintf("failed to list: %v", err)})
	} else {
		for _, pa := range peerAuths.Items {
			resource := fmt.Sprintf("PeerAuthentication %s/%s", pa.Namespace, pa.Name)
			if strings.EqualFold(pa.Spec.GetMtls().GetMode().String(), "PERMISSIVE") {
				msgs = append(msgs, analyzeMessage{levelInfo, resource,
					"mTLS mode is PERMISSIVE: plaintext connections are still accepted; switch to STRICT once all clients present certificates"})
			}
		}
	}
	return msgs
}

// analyzeCircuitBreakerPolicies reports target references to services that do not exist.
func analyzeCircuitBreakerPolicies(ctx context.Context, client kube.CLIClient, namespace string, services []corev1.Service) []analyzeMessage {
	msgs := []analyzeMessage{}
	policies, err := client.Dubbo().NetworkingV1alpha3().CircuitBreakerPolicies(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return []analyzeMessage{{levelWarning, "CircuitBreakerPolicy", fmt.Sprintf("failed to list: %v", err)}}
	}
	svcIndex := map[string]bool{}
	for _, svc := range services {
		svcIndex[svc.Namespace+"/"+svc.Name] = true
	}
	for _, policy := range policies.Items {
		resource := fmt.Sprintf("CircuitBreakerPolicy %s/%s", policy.Namespace, policy.Name)
		for i, ref := range policy.Spec.GetTargetRefs() {
			if ref.GetKind() != "Service" {
				continue
			}
			key := policy.Namespace + "/" + ref.GetName()
			if !svcIndex[key] {
				msgs = append(msgs, analyzeMessage{levelError, resource,
					fmt.Sprintf("targetRefs[%d] references service %q which does not exist", i, key)})
			}
		}
	}
	return msgs
}

func writeAnalyzeMessages(out io.Writer, msgs []analyzeMessage) {
	if len(msgs) == 0 {
		fmt.Fprintln(out, "✔ No configuration issues found.")
		return
	}
	for _, m := range msgs {
		fmt.Fprintf(out, "%s [%s] %s\n", m.Level, m.Resource, m.Message)
	}
}
