// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package horuser

import (
	"context"
	"github.com/apache/dubbo-kubernetes/app/horus/basic/config"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"time"
)

type Horuser struct {
	cc            *config.Config
	kubeClientMap map[string]*clientset.Clientset
}

func NewHoruser(c *config.Config) *Horuser {
	hr := &Horuser{
		cc:            c,
		kubeClientMap: map[string]*clientset.Clientset{},
	}
	i := 1
	n := len(c.KubeMultiple)
	for clusterName, km := range c.KubeMultiple {
		kcfg, err := k8sBuildConfig(km)
		if err != nil {
			klog.Errorf("NewHoruser k8sBuildConfig err:%v\n name:%v\n", err, clusterName)
		}
		km := clientset.NewForConfigOrDie(kcfg)
		hr.kubeClientMap[clusterName] = km
		klog.Infof("NewHoruser k8sBuildConfig success.%d/%d KubeMultipleCluster: %v", n, i, clusterName)
		i++
	}
	return hr
}

func k8sBuildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
		return cfg, err
	}
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return cfg, err
}

func (h *Horuser) GetK8sContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(h.cc.KubeTimeSecond)*time.Second)
}
