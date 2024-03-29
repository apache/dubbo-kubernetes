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

package controllers

import (
	"context"
	k8s_common "github.com/apache/dubbo-kubernetes/pkg/plugins/common/k8s"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/resources/k8s/native/api/v1alpha1"
	"github.com/go-logr/logr"
	kube_runtime "k8s.io/apimachinery/pkg/runtime"
	kube_record "k8s.io/client-go/tools/record"
	kube_ctrl "sigs.k8s.io/controller-runtime"
	kube_client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

type MedataReconciler struct {
	kube_client.Client
	kube_record.EventRecorder
	Schema            *kube_runtime.Scheme
	Log               logr.Logger
	ResourceConverter k8s_common.Converter
	SystemNamespace   string
}

func (m *MedataReconciler) Reconcile(ctx context.Context, req kube_ctrl.Request) (kube_ctrl.Result, error) {
	return kube_ctrl.Result{}, nil
}

func (m *MedataReconciler) SetupWithManager(mgr kube_ctrl.Manager, maxConcurrentReconciles int) error {
	return kube_ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxConcurrentReconciles}).
		For(&v1alpha1.MetaData{}).
		Complete(m)
}
