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

package k8s

import (
	"strings"
	"time"
)

import (
	"github.com/patrickmn/go-cache"
)

import (
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	k8s_common "github.com/apache/dubbo-kubernetes/pkg/plugins/common/k8s"
	k8s_model "github.com/apache/dubbo-kubernetes/pkg/plugins/resources/k8s/native/pkg/model"
	k8s_registry "github.com/apache/dubbo-kubernetes/pkg/plugins/resources/k8s/native/pkg/registry"
)

var _ k8s_common.Converter = &cachingConverter{}

// According to the profile, a huge amount of time is spent on marshaling of json objects.
// That's why having a cache on this execution path gives a big performance boost in Kubernetes.
type cachingConverter struct {
	SimpleConverter
	cache *cache.Cache
}

func NewCachingConverter(expirationTime time.Duration) k8s_common.Converter {
	return &cachingConverter{
		SimpleConverter: SimpleConverter{
			KubeFactory: &SimpleKubeFactory{
				KubeTypes: k8s_registry.Global(),
			},
		},
		cache: cache.New(expirationTime, time.Duration(int64(float64(expirationTime)*0.9))),
	}
}

func (c *cachingConverter) ToCoreResource(obj k8s_model.KubernetesObject, out core_model.Resource) error {
	out.SetMeta(&KubernetesMetaAdapter{ObjectMeta: *obj.GetObjectMeta(), Mesh: obj.GetMesh()})
	key := strings.Join([]string{
		obj.GetNamespace(),
		obj.GetName(),
		obj.GetResourceVersion(),
		obj.GetObjectKind().GroupVersionKind().String(),
	}, ":")
	if obj.GetResourceVersion() == "" {
		// an absent of the ResourceVersion means we decode 'obj' from webhook request,
		// all webhooks use SimpleConverter, so this is not supposed to happen
		spec, err := obj.GetSpec()
		if err != nil {
			return err
		}
		if err := out.SetSpec(spec); err != nil {
			return err
		}
	}
	if v, ok := c.cache.Get(key); ok {
		return out.SetSpec(v.(core_model.ResourceSpec))
	}
	spec, err := obj.GetSpec()
	if err != nil {
		return err
	}
	if err := out.SetSpec(spec); err != nil {
		return err
	}
	c.cache.SetDefault(key, out.GetSpec())
	return nil
}
