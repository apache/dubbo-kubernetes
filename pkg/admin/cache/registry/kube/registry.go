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
	"dubbo.apache.org/dubbo-go/v3/common"
	dubboRegistry "dubbo.apache.org/dubbo-go/v3/registry"
	"github.com/apache/dubbo-kubernetes/pkg/admin/cache"
	"github.com/apache/dubbo-kubernetes/pkg/admin/cache/registry"
	"github.com/apache/dubbo-kubernetes/pkg/admin/constant"
	"github.com/apache/dubbo-kubernetes/pkg/core/kubeclient/client"
)

func init() {
	registry.AddRegistry("kube", func(u *common.URL, kc *client.KubeClient) (registry.AdminRegistry, cache.Cache, error) {
		clusterScoped := false
		namespaces := make([]string, 0)
		if ns, ok := u.GetParams()[constant.NamespaceKey]; ok && ns[0] != constant.AnyValue {
			namespaces = append(namespaces, ns...)
		} else {
			clusterScoped = true
		}
		KubernetesCacheInstance = NewKubernetesCache(kc, clusterScoped) // init cache instance before start registry
		return NewRegistry(clusterScoped, namespaces), KubernetesCacheInstance, nil
	})
}

type Registry struct {
	clusterScoped bool
	namespaces    []string
}

func NewRegistry(clusterScoped bool, namespaces []string) *Registry {
	return &Registry{
		clusterScoped: clusterScoped,
		namespaces:    namespaces,
	}
}

func (kr *Registry) Delegate() dubboRegistry.Registry {
	return nil
}

func (kr *Registry) Subscribe() error {
	if kr.clusterScoped {
		err := KubernetesCacheInstance.startInformers()
		if err != nil {
			return err
		}
	} else {
		err := KubernetesCacheInstance.startInformers(kr.namespaces...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (kr *Registry) Destroy() error {
	KubernetesCacheInstance.stopInformers()
	return nil
}
