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
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/apache/dubbo-kubernetes/pkg/kube/inject"
	"github.com/apache/dubbo-kubernetes/pkg/kube/multicluster"
	"github.com/spf13/cobra"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/yaml"
)

type remoteSecretArgs struct {
	clusterName string
	name        string
	namespace   string
	kubeconfig  string
	context     string
}

type remoteManifestArgs struct {
	clusterName  string
	webhookURL   string
	xdsAddress   string
	caAddress    string
	caBundleFile string
	revision     string
}

type eastWestGatewayArgs struct {
	name        string
	namespace   string
	serviceType string
	port        int32
	targetPort  int
	nodePort    int32
	xdsAddress  string
}

func MulticlusterCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "multicluster",
		Short: "Manage remote clusters for dubbod",
	}
	command.AddCommand(createRemoteSecretCmd())
	command.AddCommand(generateRemoteManifestCmd())
	command.AddCommand(generateEastWestGatewayCmd())
	return command
}

func createRemoteSecretCmd() *cobra.Command {
	args := &remoteSecretArgs{
		namespace: "dubbo-system",
	}
	command := &cobra.Command{
		Use:   "create-remote-secret",
		Short: "Generate a remote-cluster Secret manifest",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			kubeconfig, err := remoteKubeconfig(args.kubeconfig, args.context)
			if err != nil {
				return err
			}
			secret, err := buildRemoteClusterSecret(*args, kubeconfig)
			if err != nil {
				return err
			}
			raw, err := yaml.Marshal(secret)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(raw)
			return err
		},
	}
	flags := command.Flags()
	flags.StringVar(&args.clusterName, "cluster-name", "", "Unique remote cluster name")
	flags.StringVar(&args.name, "name", "", "Secret name; defaults to dubbo-remote-<cluster-name>")
	flags.StringVarP(&args.namespace, "namespace", "n", args.namespace, "Namespace where dubbod reads remote cluster Secrets")
	flags.StringVar(&args.kubeconfig, "kubeconfig", "", "Remote cluster kubeconfig path")
	flags.StringVar(&args.context, "context", "", "Remote cluster kubeconfig context")
	_ = command.MarkFlagRequired("cluster-name")
	return command
}

func generateRemoteManifestCmd() *cobra.Command {
	args := &remoteManifestArgs{
		revision: "default",
	}
	command := &cobra.Command{
		Use:   "generate-remote-manifest",
		Short: "Generate remote-cluster injection webhook manifest",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			caBundle, err := os.ReadFile(args.caBundleFile)
			if err != nil {
				return fmt.Errorf("read CA bundle: %w", err)
			}
			manifest, err := buildRemoteWebhookManifest(*args, caBundle)
			if err != nil {
				return err
			}
			raw, err := yaml.Marshal(manifest)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(raw)
			return err
		},
	}
	flags := command.Flags()
	flags.StringVar(&args.clusterName, "cluster-name", "", "Unique remote cluster name")
	flags.StringVar(&args.webhookURL, "webhook-url", "", "Externally reachable dubbod webhook base URL")
	flags.StringVar(&args.xdsAddress, "xds-address", "", "Externally reachable dubbod xDS address")
	flags.StringVar(&args.caAddress, "ca-address", "", "Externally reachable dubbod CA address")
	flags.StringVar(&args.caBundleFile, "ca-bundle-file", "", "PEM CA bundle file for webhook TLS verification")
	flags.StringVar(&args.revision, "revision", args.revision, "Revision label used by the remote injector")
	for _, flag := range []string{"cluster-name", "webhook-url", "xds-address", "ca-address", "ca-bundle-file"} {
		_ = command.MarkFlagRequired(flag)
	}
	return command
}

func generateEastWestGatewayCmd() *cobra.Command {
	args := &eastWestGatewayArgs{
		name:        "dubbod-eastwest-gateway",
		namespace:   "dubbo-system",
		serviceType: "LoadBalancer",
		port:        15443,
		targetPort:  inject.ProxylessGRPCInboundPort,
	}
	command := &cobra.Command{
		Use:   "generate-eastwest-gateway",
		Short: "Generate an east-west Gateway manifest for a workload cluster",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			gw, err := buildEastWestGatewayManifest(*args)
			if err != nil {
				return err
			}
			raw, err := yaml.Marshal(gw)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(raw)
			return err
		},
	}
	flags := command.Flags()
	flags.StringVar(&args.name, "name", args.name, "Gateway name")
	flags.StringVarP(&args.namespace, "namespace", "n", args.namespace, "Gateway namespace")
	flags.StringVar(&args.serviceType, "service-type", args.serviceType, "Managed gateway Service type")
	flags.Int32Var(&args.port, "port", args.port, "Externally reachable east-west Gateway port")
	flags.IntVar(&args.targetPort, "target-port", args.targetPort, "Injected grpc-inbound target port used by the managed gateway Service")
	flags.Int32Var(&args.nodePort, "node-port", 0, "Optional managed gateway Service nodePort")
	flags.StringVar(&args.xdsAddress, "xds-address", "", "ADS address used by the remote dxgate data plane")
	_ = command.MarkFlagRequired("xds-address")
	return command
}

func remoteKubeconfig(path, contextName string) ([]byte, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if path != "" {
		rules.ExplicitPath = path
	}
	rawConfig, err := rules.Load()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}
	if contextName == "" {
		contextName = rawConfig.CurrentContext
	}
	context, ok := rawConfig.Contexts[contextName]
	if !ok {
		return nil, fmt.Errorf("context %q not found", contextName)
	}
	minimal := clientcmdapi.Config{
		Kind:           rawConfig.Kind,
		APIVersion:     rawConfig.APIVersion,
		CurrentContext: contextName,
		Contexts:       map[string]*clientcmdapi.Context{contextName: context.DeepCopy()},
		Clusters:       map[string]*clientcmdapi.Cluster{},
		AuthInfos:      map[string]*clientcmdapi.AuthInfo{},
	}
	if clusterConfig, ok := rawConfig.Clusters[context.Cluster]; ok {
		minimal.Clusters[context.Cluster] = clusterConfig.DeepCopy()
	}
	if authInfo, ok := rawConfig.AuthInfos[context.AuthInfo]; ok {
		minimal.AuthInfos[context.AuthInfo] = authInfo.DeepCopy()
	}
	return clientcmd.Write(minimal)
}

func buildEastWestGatewayManifest(args eastWestGatewayArgs) (*gatewayv1.Gateway, error) {
	if strings.TrimSpace(args.name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	if strings.TrimSpace(args.namespace) == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	if args.port <= 0 || args.port > 65535 {
		return nil, fmt.Errorf("port must be between 1 and 65535")
	}
	if args.targetPort <= 0 || args.targetPort > 65535 {
		return nil, fmt.Errorf("target-port must be between 1 and 65535")
	}
	if args.nodePort < 0 || args.nodePort > 65535 {
		return nil, fmt.Errorf("node-port must be between 0 and 65535")
	}
	if strings.TrimSpace(args.xdsAddress) == "" {
		return nil, fmt.Errorf("xds-address is required")
	}
	annotations := map[string]string{
		"gateway.dubbo.apache.org/eastwest":     "true",
		"gateway.dubbo.apache.org/service-type": args.serviceType,
		"gateway.dubbo.apache.org/target-port":  fmt.Sprintf("%d", args.targetPort),
		"gateway.dubbo.apache.org/xds-address":  args.xdsAddress,
	}
	if args.nodePort > 0 {
		annotations["gateway.dubbo.apache.org/node-port"] = fmt.Sprintf("%d", args.nodePort)
	}
	return &gatewayv1.Gateway{
		TypeMeta: metav1.TypeMeta{
			APIVersion: gatewayv1.GroupVersion.String(),
			Kind:       "Gateway",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      args.name,
			Namespace: args.namespace,
			Labels: map[string]string{
				"app":                               "dubbod-eastwest-gateway",
				"gateway.dubbo.apache.org/eastwest": "true",
			},
			Annotations: annotations,
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: gatewayv1.ObjectName("dubbo"),
			Listeners: []gatewayv1.Listener{{
				Name:     gatewayv1.SectionName("http-eastwest"),
				Protocol: gatewayv1.HTTPProtocolType,
				Port:     args.port,
				AllowedRoutes: &gatewayv1.AllowedRoutes{
					Namespaces: &gatewayv1.RouteNamespaces{
						From: ptrGatewayFrom(gatewayv1.NamespacesFromAll),
					},
				},
			}},
		},
	}, nil
}

func ptrGatewayFrom(value gatewayv1.FromNamespaces) *gatewayv1.FromNamespaces {
	return &value
}

func buildRemoteClusterSecret(args remoteSecretArgs, kubeconfig []byte) (*corev1.Secret, error) {
	if strings.TrimSpace(args.clusterName) == "" {
		return nil, fmt.Errorf("cluster-name is required")
	}
	if args.name == "" {
		args.name = "dubbo-remote-" + args.clusterName
	}
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      args.name,
			Namespace: args.namespace,
			Labels: map[string]string{
				multicluster.RemoteClusterSecretLabel: multicluster.RemoteClusterSecretLabelValue,
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			multicluster.RemoteClusterSecretClusterNameKey: []byte(args.clusterName),
			multicluster.RemoteClusterSecretKubeconfigKey:  kubeconfig,
		},
	}, nil
}

func buildRemoteWebhookManifest(args remoteManifestArgs, caBundle []byte) (*admissionregistrationv1.MutatingWebhookConfiguration, error) {
	if strings.TrimSpace(args.clusterName) == "" {
		return nil, fmt.Errorf("cluster-name is required")
	}
	webhookURL, err := remoteWebhookURL(args.webhookURL, args.clusterName, args.xdsAddress, args.caAddress)
	if err != nil {
		return nil, err
	}
	sideEffects := admissionregistrationv1.SideEffectClassNone
	failurePolicy := admissionregistrationv1.Fail
	reinvocationPolicy := admissionregistrationv1.NeverReinvocationPolicy
	scope := admissionregistrationv1.NamespacedScope
	clientConfig := admissionregistrationv1.WebhookClientConfig{
		URL:      &webhookURL,
		CABundle: caBundle,
	}
	return &admissionregistrationv1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: admissionregistrationv1.SchemeGroupVersion.String(),
			Kind:       "MutatingWebhookConfiguration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "dubbo-remote-proxyless-injector-" + args.clusterName,
			Labels: map[string]string{
				"app":                  "dubbod",
				"dubbo.apache.org/rev": args.revision,
			},
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			remoteWebhook("rev.namespace.remote.proxyless-injector.dubbo.apache.org", clientConfig, &failurePolicy, &sideEffects, &reinvocationPolicy, &scope,
				&metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "dubbo.apache.org/rev", Operator: metav1.LabelSelectorOpIn, Values: []string{args.revision}},
					{Key: "dubbo-injection", Operator: metav1.LabelSelectorOpDoesNotExist},
				}},
				&metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "proxyless.dubbo.apache.org/inject", Operator: metav1.LabelSelectorOpNotIn, Values: []string{"false"}},
				}},
			),
			remoteWebhook("rev.object.remote.proxyless-injector.dubbo.apache.org", clientConfig, &failurePolicy, &sideEffects, &reinvocationPolicy, &scope,
				&metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "dubbo.apache.org/rev", Operator: metav1.LabelSelectorOpDoesNotExist},
					{Key: "dubbo-injection", Operator: metav1.LabelSelectorOpDoesNotExist},
				}},
				&metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "proxyless.dubbo.apache.org/inject", Operator: metav1.LabelSelectorOpNotIn, Values: []string{"false"}},
					{Key: "dubbo.apache.org/rev", Operator: metav1.LabelSelectorOpIn, Values: []string{args.revision}},
				}},
			),
			remoteWebhook("namespace.remote.proxyless-injector.dubbo.apache.org", clientConfig, nil, &sideEffects, &reinvocationPolicy, &scope,
				&metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "dubbo-injection", Operator: metav1.LabelSelectorOpIn, Values: []string{"enabled"}},
				}},
				&metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "proxyless.dubbo.apache.org/inject", Operator: metav1.LabelSelectorOpNotIn, Values: []string{"false"}},
				}},
			),
			remoteWebhook("object.remote.proxyless-injector.dubbo.apache.org", clientConfig, nil, &sideEffects, &reinvocationPolicy, &scope,
				&metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "dubbo-injection", Operator: metav1.LabelSelectorOpDoesNotExist},
				}},
				&metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "proxyless.dubbo.apache.org/inject", Operator: metav1.LabelSelectorOpIn, Values: []string{"true"}},
				}},
			),
		},
	}, nil
}

func remoteWebhook(name string, clientConfig admissionregistrationv1.WebhookClientConfig, failurePolicy *admissionregistrationv1.FailurePolicyType,
	sideEffects *admissionregistrationv1.SideEffectClass, reinvocationPolicy *admissionregistrationv1.ReinvocationPolicyType,
	scope *admissionregistrationv1.ScopeType, namespaceSelector, objectSelector *metav1.LabelSelector,
) admissionregistrationv1.MutatingWebhook {
	return admissionregistrationv1.MutatingWebhook{
		Name:                    name,
		AdmissionReviewVersions: []string{"v1"},
		ClientConfig:            clientConfig,
		FailurePolicy:           failurePolicy,
		NamespaceSelector:       namespaceSelector,
		ObjectSelector:          objectSelector,
		ReinvocationPolicy:      reinvocationPolicy,
		Rules: []admissionregistrationv1.RuleWithOperations{{
			Operations: []admissionregistrationv1.OperationType{
				admissionregistrationv1.Create,
			},
			Rule: admissionregistrationv1.Rule{
				APIGroups:   []string{""},
				APIVersions: []string{"v1"},
				Resources:   []string{"pods"},
				Scope:       scope,
			},
		}, {
			Operations: []admissionregistrationv1.OperationType{
				admissionregistrationv1.Create,
				admissionregistrationv1.Update,
			},
			Rule: admissionregistrationv1.Rule{
				APIGroups:   []string{""},
				APIVersions: []string{"v1"},
				Resources:   []string{"services"},
				Scope:       scope,
			},
		}},
		SideEffects: sideEffects,
	}
}

func remoteWebhookURL(rawURL, clusterName, xdsAddress, caAddress string) (string, error) {
	if strings.TrimSpace(xdsAddress) == "" || strings.TrimSpace(caAddress) == "" {
		return "", fmt.Errorf("xds-address and ca-address are required")
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("webhook-url must be an absolute URL")
	}
	path := injectPath(clusterName, xdsAddress, caAddress)
	u.Path = strings.TrimSuffix(u.Path, "/") + path
	return u.String(), nil
}

func injectPath(clusterName, xdsAddress, caAddress string) string {
	segments := []string{
		"DUBBO_META_CLUSTER_ID", clusterName,
		"XDS_ADDRESS", xdsAddress,
		"CA_ADDRESS", caAddress,
	}
	for i := 1; i < len(segments); i += 2 {
		segments[i] = strings.ReplaceAll(segments[i], "/", "--slash--")
	}
	return "/inject/" + strings.Join(segments, "/")
}

func encodedSecretData(secret *corev1.Secret, key string) string {
	return base64.StdEncoding.EncodeToString(secret.Data[key])
}
