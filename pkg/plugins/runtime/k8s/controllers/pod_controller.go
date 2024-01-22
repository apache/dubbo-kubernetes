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
)

import (
	"github.com/go-logr/logr"

	kube_core "k8s.io/api/core/v1"

	kube_apierrs "k8s.io/apimachinery/pkg/api/errors"
	kube_runtime "k8s.io/apimachinery/pkg/runtime"
	kube_types "k8s.io/apimachinery/pkg/types"

	kube_record "k8s.io/client-go/tools/record"

	kube_ctrl "sigs.k8s.io/controller-runtime"
	kube_client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	kube_handler "sigs.k8s.io/controller-runtime/pkg/handler"
	kube_reconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

import (
	k8s_common "github.com/apache/dubbo-kubernetes/pkg/plugins/common/k8s"
)

const (
	// CreateddubboDataplaneReason is added to an event when
	// a new Dataplane is successfully created.
	CreateddubboDataplaneReason = "CreateddubboDataplane"
	// UpdateddubboDataplaneReason is added to an event when
	// an existing Dataplane is successfully updated.
	UpdateddubboDataplaneReason = "UpdateddubboDataplane"
	// FailedToGeneratedubboDataplaneReason is added to an event when
	// a Dataplane cannot be generated or is not valid.
	FailedToGeneratedubboDataplaneReason = "FailedToGeneratedubboDataplane"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	kube_client.Client
	kube_record.EventRecorder
	Scheme                       *kube_runtime.Scheme
	Log                          logr.Logger
	PodConverter                 PodConverter
	ResourceConverter            k8s_common.Converter
	SystemNamespace              string
	IgnoredServiceSelectorLabels []string
}

func (r *PodReconciler) Reconcile(ctx context.Context, req kube_ctrl.Request) (kube_ctrl.Result, error) {
	log := r.Log.WithValues("pod", req.NamespacedName)
	log.V(1).Info("reconcile")

	// Fetch the Pod instance
	pod := &kube_core.Pod{}
	if err := r.Get(ctx, req.NamespacedName, pod); err != nil {
		if kube_apierrs.IsNotFound(err) {
			log.V(1).Info("pod not found. Skipping")
			return kube_ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch Pod")
		return kube_ctrl.Result{}, err
	}

	return kube_ctrl.Result{}, nil
}

func (r *PodReconciler) SetupWithManager(mgr kube_ctrl.Manager, maxConcurrentReconciles int) error {
	return kube_ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxConcurrentReconciles}).
		For(&kube_core.Pod{}).
		// on Service update reconcile affected Pods (all Pods selected by this service)
		Watches(&kube_core.Service{}, kube_handler.EnqueueRequestsFromMapFunc(ServiceToPodsMapper(r.Log, mgr.GetClient()))).
		Complete(r)
}

func ServiceToPodsMapper(l logr.Logger, client kube_client.Client) kube_handler.MapFunc {
	l = l.WithName("service-to-pods-mapper")
	return func(ctx context.Context, obj kube_client.Object) []kube_reconcile.Request {
		// List Pods in the same namespace as a Service
		pods := &kube_core.PodList{}
		if err := client.List(ctx, pods, kube_client.InNamespace(obj.GetNamespace()), kube_client.MatchingLabels(obj.(*kube_core.Service).Spec.Selector)); err != nil {
			l.WithValues("service", obj.GetName()).Error(err, "failed to fetch Pods")
			return nil
		}
		var req []kube_reconcile.Request
		for _, pod := range pods.Items {
			req = append(req, kube_reconcile.Request{
				NamespacedName: kube_types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name},
			})
		}
		return req
	}
}
