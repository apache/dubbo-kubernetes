/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package kube

import (
	"github.com/apache/dubbo-kubernetes/sail/pkg/features"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

func DefaultRestConfig(kubeconfig, configContext string, fns ...func(config *rest.Config)) (*rest.Config, error) {
	config, err := BuildClientConfig(kubeconfig, configContext)
	if err != nil {
		return nil, err
	}
	for _, fn := range fns {
		fn(config)
	}
	return config, nil
}

func BuildClientCmd(kubeconfig, context string, overrides ...func(configOverrides *clientcmd.ConfigOverrides)) clientcmd.ClientConfig {
	if kubeconfig != "" {
		info, err := os.Stat(kubeconfig)
		if err != nil || info.Size() == 0 {
			kubeconfig = ""
		}
	}
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	loadingRules.ExplicitPath = kubeconfig
	configOverrides := &clientcmd.ConfigOverrides{
		ClusterDefaults: clientcmd.ClusterDefaults,
		CurrentContext:  context,
	}
	for _, fn := range overrides {
		fn(configOverrides)
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
}

func BuildClientConfig(kubeconfig, context string) (*rest.Config, error) {
	c, err := BuildClientCmd(kubeconfig, context).ClientConfig()
	if err != nil {
		return nil, err
	}
	return SetRestDefaults(c), nil
}

func SetRestDefaults(config *rest.Config) *rest.Config {
	if config.GroupVersion == nil || config.GroupVersion.Empty() {
		config.GroupVersion = &corev1.SchemeGroupVersion
	}
	if len(config.APIPath) == 0 {
		if len(config.GroupVersion.Group) == 0 {
			config.APIPath = "/api"
		} else {
			config.APIPath = "/apis"
		}
	}
	if len(config.ContentType) == 0 {
		if features.KubernetesClientContentType == "json" {
			config.ContentType = runtime.ContentTypeJSON
		} else {
			config.AcceptContentTypes = runtime.ContentTypeProtobuf + "," + runtime.ContentTypeJSON
			config.ContentType = runtime.ContentTypeJSON
		}
	}
	if config.NegotiatedSerializer == nil {
		config.NegotiatedSerializer = serializer.NewCodecFactory(nil).WithoutConversion()
	}
	return config
}
