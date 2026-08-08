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

package activation

import (
	"strings"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kclient"
	"github.com/apache/dubbo-kubernetes/pkg/log"
	clientnetworking "github.com/kdubbo/client-go/pkg/apis/networking/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

var logger = log.RegisterScope("activation", "Service activation policies")

// resyncInterval re-evaluates every policy periodically. Two of the four
// conditions are also re-evaluated periodically as a safety net for missed
// informer edges.
const resyncInterval = 30 * time.Second

// Controller keeps ServiceActivationPolicy status in step with what the mesh
// can actually do for that policy.
//
// It publishes status only. Replica counts belong to the autoscaler the policy
// references, and writing them here would put two controllers on the same
// field.
type Controller struct {
	policies kclient.Client[*clientnetworking.ServiceActivationPolicy]
	services kclient.Client[*corev1.Service]
	queue    controllers.Queue

	evaluator PolicyEvaluator
	readiness *clusterReadiness
}

// NewController wires policy status to cluster-visible autoscaler and Gateway
// state, so every HA replica evaluates the same facts.
func NewController(client kube.Client) *Controller {
	readiness := newClusterReadiness(client)
	c := &Controller{
		policies:  kclient.New[*clientnetworking.ServiceActivationPolicy](client),
		services:  kclient.New[*corev1.Service](client),
		readiness: readiness,
	}
	c.evaluator = PolicyEvaluator{
		Services:  c,
		Scaler:    readiness,
		Activator: readiness,
	}

	c.queue = controllers.NewQueue("service activation policy",
		controllers.WithReconciler(c.Reconcile),
		controllers.WithMaxAttempts(5))

	c.policies.AddEventHandler(controllers.EventHandler[*clientnetworking.ServiceActivationPolicy]{
		AddFunc: func(policy *clientnetworking.ServiceActivationPolicy) {
			c.queue.AddObject(policy)
		},
		UpdateFunc: func(oldPolicy, newPolicy *clientnetworking.ServiceActivationPolicy) {
			// Do not feed our status writes straight back into the queue.
			// ScaledObject and Gateway informers drive runtime convergence.
			if oldPolicy.GetGeneration() != newPolicy.GetGeneration() {
				c.queue.AddObject(newPolicy)
			}
		},
		DeleteFunc: func(policy *clientnetworking.ServiceActivationPolicy) {
			c.queue.AddObject(policy)
		},
	})
	// A policy is only accepted once its target Service exists, so a Service
	// appearing later has to re-open the policies that were rejected for it.
	c.services.AddEventHandler(controllers.ObjectHandler(func(o controllers.Object) {
		for _, policy := range c.policies.List(o.GetNamespace(), klabels.Everything()) {
			c.queue.AddObject(policy)
		}
	}))
	readiness.AddEventHandlers(
		func(namespace, name string) {
			for _, policy := range c.policies.List(namespace, klabels.Everything()) {
				ref := policy.Spec.GetAutoscalerRef()
				if ref != nil &&
					ref.GetName() == name &&
					strings.EqualFold(ref.GetGroup(), scaledObjectGVR.Group) &&
					strings.EqualFold(ref.GetKind(), "ScaledObject") {
					c.queue.AddObject(policy)
				}
			}
		},
		func(namespace string) {
			for _, policy := range c.policies.List(namespace, klabels.Everything()) {
				c.queue.AddObject(policy)
			}
		},
		func(className string) {
			namespaces := map[string]struct{}{}
			for _, gateway := range readiness.gateways.List(metav1.NamespaceAll, klabels.Everything()) {
				if string(gateway.Spec.GatewayClassName) == className {
					namespaces[gateway.GetNamespace()] = struct{}{}
				}
			}
			for namespace := range namespaces {
				for _, policy := range c.policies.List(namespace, klabels.Everything()) {
					c.queue.AddObject(policy)
				}
			}
		},
	)

	return c
}

// HasService satisfies ServiceLookup from the informer cache.
func (c *Controller) HasService(namespace, name string) bool {
	return c.services.Get(name, namespace) != nil
}

func (c *Controller) Run(stop <-chan struct{}) {
	kube.WaitForCacheSync(
		"activation controller",
		stop,
		c.policies.HasSynced,
		c.services.HasSynced,
		c.readiness.HasSynced,
	)

	go c.resync(stop)

	c.queue.Run(stop)
	controllers.ShutdownAll(c.policies, c.services)
	c.readiness.ShutdownHandlers()
}

// resync re-queues every policy on a tick, picking up scaler and gateway
// changes that Kubernetes never reports.
func (c *Controller) resync(stop <-chan struct{}) {
	ticker := time.NewTicker(resyncInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			for _, policy := range c.policies.List(metav1.NamespaceAll, klabels.Everything()) {
				c.queue.AddObject(policy)
			}
		}
	}
}

func (c *Controller) Reconcile(key types.NamespacedName) error {
	policy := c.policies.Get(key.Name, key.Namespace)
	if policy == nil {
		// Deleted; nothing to publish.
		return nil
	}

	conditions := c.evaluator.Evaluate(policy)
	if SameConditions(policy.Status.GetConditions(), conditions) {
		// Writing an unchanged status would feed the resync tick back into
		// itself and turn a quiet cluster into a steady write load.
		return nil
	}

	updated := policy.DeepCopy()
	updated.Status.Conditions = conditions
	if _, err := c.policies.UpdateStatus(updated); err != nil {
		return err
	}

	logger.Debugf("updated %s/%s: %s", key.Namespace, key.Name, Summary(conditions))
	return nil
}
