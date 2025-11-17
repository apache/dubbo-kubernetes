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

package krt

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/util/ptr"
	"reflect"
	"strconv"
	"strings"
	"time"

	acmetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
)

type ObjectDecorator interface {
	GetObjectKeyable() any
}

// Named is a convenience struct. It is ideal to be embedded into a type that has a name and namespace,
// and will automatically implement the various interfaces to return the name, namespace, and a key based on these two.
type Named struct {
	Name, Namespace string
}

func (n Named) ResourceName() string {
	return n.Namespace + "/" + n.Name
}

func (n Named) GetName() string {
	return n.Name
}

func (n Named) GetNamespace() string {
	return n.Namespace
}

// GetKey returns the key for the provided object.
// If there is none, this will panic.
func GetKey[O any](a O) string {
	as, ok := any(a).(string)
	if ok {
		return as
	}
	ao, ok := any(a).(controllers.Object)
	if ok {
		k, _ := cache.MetaNamespaceKeyFunc(ao)
		return k
	}
	ac, ok := any(a).(config.Config)
	if ok {
		return keyFunc(ac.Name, ac.Namespace)
	}
	acp, ok := any(a).(*config.Config)
	if ok {
		return keyFunc(acp.Name, acp.Namespace)
	}
	arn, ok := any(a).(ResourceNamer)
	if ok {
		return arn.ResourceName()
	}
	auid, ok := any(a).(uidable)
	if ok {
		return strconv.FormatUint(uint64(auid.uid()), 10)
	}

	akclient, ok := any(a).(kube.Client)
	if ok {
		return string(akclient.ClusterID())
	}

	aobjDecorator, ok := any(a).(ObjectDecorator)
	if ok {
		return GetKey(aobjDecorator.GetObjectKeyable())
	}

	ack := GetApplyConfigKey(a)
	if ack != nil {
		return *ack
	}
	// panic(fmt.Sprintf("Cannot get Key, got %T", a))
	return fmt.Sprintf("Cannot get Key, got %T", a)
}

// GetApplyConfigKey returns the key for the ApplyConfig.
// If there is none, this will return nil.
func GetApplyConfigKey[O any](a O) *string {
	// Reflection is expensive; short circuit here
	if !strings.HasSuffix(ptr.TypeName[O](), "ApplyConfiguration") {
		return nil
	}
	val := reflect.ValueOf(a)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil
	}

	specField := val.FieldByName("ObjectMetaApplyConfiguration")
	if !specField.IsValid() {
		return nil
	}
	meta := specField.Interface().(*acmetav1.ObjectMetaApplyConfiguration)
	if meta.Namespace != nil && len(*meta.Namespace) > 0 {
		return ptr.Of(*meta.Namespace + "/" + *meta.Name)
	}
	return meta.Name
}

func getTypedKey[O any](a O) Key[O] {
	return Key[O](GetKey(a))
}

// keyFunc is the internal API key function that returns "namespace"/"name" or
// "name" if "namespace" is empty
func keyFunc(name, namespace string) string {
	if len(namespace) == 0 {
		return name
	}
	return namespace + "/" + name
}

func waitForCacheSync(name string, stop <-chan struct{}, collections ...<-chan struct{}) (r bool) {
	t := time.NewTicker(time.Second * 5)
	defer t.Stop()
	t0 := time.Now()
	defer func() {
		if r {
			log.Infof("sync complete: name=%s, time=%v", name, time.Since(t0))
		} else {
			log.Infof("sync failed: name=%s, time=%v", name, time.Since(t0))
		}
	}()
	for _, col := range collections {
		for {
			select {
			case <-t.C:
				log.Infof("waiting for sync...: name=%s, time=%v\n", name, time.Since(t0))
				continue
			case <-stop:
				return false
			case <-col:
			}
			break
		}
	}
	return true
}
