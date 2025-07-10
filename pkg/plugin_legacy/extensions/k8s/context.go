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
	"context"
)

import (
	kube_ctrl "sigs.k8s.io/controller-runtime"
	kube_client "sigs.k8s.io/controller-runtime/pkg/client"
)

import (
	k8s_common "github.com/apache/dubbo-kubernetes/pkg/plugins/common/k8s"
)

type managerKey struct{}

func NewManagerContext(ctx context.Context, manager kube_ctrl.Manager) context.Context {
	return context.WithValue(ctx, managerKey{}, manager)
}

func FromManagerContext(ctx context.Context) (kube_ctrl.Manager, bool) {
	manager, ok := ctx.Value(managerKey{}).(kube_ctrl.Manager)
	return manager, ok
}

// One instance of Converter needs to be shared across resource plugin and runtime
// plugin if CachedConverter is used, only one instance is created, otherwise we would
// have all cached resources in the memory twice.

type converterKey struct{}

func NewResourceConverterContext(ctx context.Context, converter k8s_common.Converter) context.Context {
	return context.WithValue(ctx, converterKey{}, converter)
}

func FromResourceConverterContext(ctx context.Context) (k8s_common.Converter, bool) {
	converter, ok := ctx.Value(converterKey{}).(k8s_common.Converter)
	return converter, ok
}

type secretClient struct{}

func NewSecretClientContext(ctx context.Context, client kube_client.Client) context.Context {
	return context.WithValue(ctx, secretClient{}, client)
}

func FromSecretClientContext(ctx context.Context) (kube_client.Client, bool) {
	client, ok := ctx.Value(secretClient{}).(kube_client.Client)
	return client, ok
}

type compositeValidatorKey struct{}

func NewCompositeValidatorContext(ctx context.Context, compositeValidator *k8s_common.CompositeValidator) context.Context {
	return context.WithValue(ctx, compositeValidatorKey{}, compositeValidator)
}

func FromCompositeValidatorContext(ctx context.Context) (*k8s_common.CompositeValidator, bool) {
	validator, ok := ctx.Value(compositeValidatorKey{}).(*k8s_common.CompositeValidator)
	return validator, ok
}
