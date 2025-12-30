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

package status

import (
	"context"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvk"
)

type Resource struct {
	schema.GroupVersionResource
	Namespace  string
	Name       string
	Generation string
}

func (r Resource) String() string {
	return strings.Join([]string{r.Group, r.Version, r.GroupVersionResource.Resource, r.Namespace, r.Name, r.Generation}, "/")
}

func (r *Resource) ToModelKey() string {
	gk, ok := gvk.FromGVR(r.GroupVersionResource)
	if !ok {
		return ""
	}
	return config.Key(
		gk.Group, gk.Version, gk.Kind,
		r.Name, r.Namespace)
}

func ResourceFromModelConfig(c config.Config) Resource {
	gvr, ok := gvk.ToGVR(c.GroupVersionKind)
	if !ok {
		return Resource{}
	}
	return Resource{
		GroupVersionResource: gvr,
		Namespace:            c.Namespace,
		Name:                 c.Name,
		Generation:           strconv.FormatInt(c.Generation, 10),
	}
}

func GetStatusManipulator(in any) (out Manipulator) {
	return &NopStatusManipulator{in}
}

func NewIstioContext(stop <-chan struct{}) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-stop
		cancel()
	}()
	return ctx
}
