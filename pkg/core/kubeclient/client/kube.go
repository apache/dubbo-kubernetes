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

package client

import (
	"os"
	"path/filepath"
	"reflect"
	"time"

	kubeExtClient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"

	dubbo_cp "github.com/apache/dubbo-kubernetes/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type KubeClient struct {
	kubernetes.Interface

	kubernetesClientSet kubernetes.Interface
	kubeInformer        informers.SharedInformerFactory
	kubeConfig          *rest.Config
	extSet              kubeExtClient.Interface
}

func NewKubeClient() *KubeClient {
	return &KubeClient{}
}

func (c *KubeClient) Ext() kubeExtClient.Interface {
	return c.extSet
}

func (k *KubeClient) GetKubeConfig() *rest.Config {
	return k.kubeConfig
}

func (k *KubeClient) GetKubernetesClientSet() kubernetes.Interface {
	return k.kubernetesClientSet
}

// nolint
func (k *KubeClient) Start(stop <-chan struct{}) error {
	return nil
}

func (k *KubeClient) NeedLeaderElection() bool {
	return false
}

func (k *KubeClient) Init(options *dubbo_cp.Config) bool {
	config, err := rest.InClusterConfig()
	kubeconfig := options.Runtime.Kubernetes.ClientConfig.KubeFileConfig
	if err != nil {
		logger.Sugar().Infof("Failed to load config from Pod. Will fall back to kube config file.")
		// Read kubeconfig from command line
		if len(kubeconfig) <= 0 {
			// Read kubeconfig from env
			kubeconfig = os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
			if len(kubeconfig) <= 0 {
				// Read kubeconfig from home dir
				if home := homedir.HomeDir(); home != "" {
					kubeconfig = filepath.Join(home, ".kube", "config")
				}
			}
		}
		// use the current context in kubeconfig
		logger.Sugar().Infof("Read kubeconfig from %s", kubeconfig)
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			logger.Sugar().Warnf("Failed to load config from kube config file.")
			return false
		}
	}

	// set qps and burst for rest config
	config.QPS = float32(options.Runtime.Kubernetes.ClientConfig.Qps)
	config.Burst = options.Runtime.Kubernetes.ClientConfig.BurstQps
	k.kubeConfig = config
	// creates the client
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Sugar().Warnf("Failed to create clientgen to kubernetes. " + err.Error())
		return false
	}
	if err != nil {
		logger.Sugar().Warnf("Failed to create clientgen to kubernetes. " + err.Error())
		return false
	}
	k.kubernetesClientSet = clientSet
	ext, err := kubeExtClient.NewForConfig(config)
	if err != nil {
		logger.Sugar().Warnf("Failed to create kubeExtClient to kubernetes. " + err.Error())
		return false
	}
	k.extSet = ext
	return true
}

type reflectInformerSync interface {
	WaitForCacheSync(stopCh <-chan struct{}) map[reflect.Type]bool
}

// Wait for cache sync immediately, rather than with 100ms delay which slows tests
// See https://github.com/kubernetes/kubernetes/issues/95262#issuecomment-703141573
// nolint
func fastWaitForCacheSync(informerFactory reflectInformerSync) {
	returnImmediately := make(chan struct{})
	close(returnImmediately)
	_ = wait.PollImmediate(time.Microsecond, wait.ForeverTestTimeout, func() (bool, error) {
		for _, synced := range informerFactory.WaitForCacheSync(returnImmediately) {
			if !synced {
				return false, nil
			}
		}
		return true, nil
	})
}
