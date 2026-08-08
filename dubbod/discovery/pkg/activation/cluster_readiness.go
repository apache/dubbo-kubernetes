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

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/features"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/gvr"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kclient"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kubetypes"
	clientnetworking "github.com/kdubbo/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var scaledObjectGVR = schema.GroupVersionResource{
	Group: "keda.sh", Version: "v1alpha1", Resource: "scaledobjects",
}

// clusterReadiness derives policy status from objects all dubbod replicas see.
// Process-local streams are deliberately not used: in HA, KEDA and gateways
// can connect to different replicas, so no replica has a complete view.
type clusterReadiness struct {
	scaledObjects  kclient.Informer[controllers.Object]
	gateways       kclient.Informer[*gatewayv1.Gateway]
	gatewayClasses kclient.Informer[*gatewayv1.GatewayClass]
}

func newClusterReadiness(client kube.Client) *clusterReadiness {
	filter := kclient.Filter{ObjectFilter: client.ObjectFilter()}
	return &clusterReadiness{
		scaledObjects: kclient.NewDelayedInformer[controllers.Object](
			client, scaledObjectGVR, kubetypes.DynamicInformer, filter,
		),
		gateways: kclient.NewDelayedInformer[*gatewayv1.Gateway](
			client, gvr.KubernetesGateway, kubetypes.StandardInformer, filter,
		),
		gatewayClasses: kclient.NewDelayedInformer[*gatewayv1.GatewayClass](
			client, gvr.GatewayClass, kubetypes.StandardInformer, filter,
		),
	}
}

func (r *clusterReadiness) AddEventHandlers(
	scaledObjectChanged func(namespace, name string),
	gatewayChanged func(namespace string),
	gatewayClassChanged func(name string),
) {
	r.scaledObjects.AddEventHandler(controllers.ObjectHandler(func(object controllers.Object) {
		scaledObjectChanged(object.GetNamespace(), object.GetName())
	}))
	r.gateways.AddEventHandler(controllers.ObjectHandler(func(object controllers.Object) {
		gatewayChanged(object.GetNamespace())
	}))
	r.gatewayClasses.AddEventHandler(controllers.ObjectHandler(func(object controllers.Object) {
		gatewayClassChanged(object.GetName())
	}))
}

func (r *clusterReadiness) HasSynced() bool {
	return r.scaledObjects.HasSynced() &&
		r.gateways.HasSynced() &&
		r.gatewayClasses.HasSynced()
}

func (r *clusterReadiness) ShutdownHandlers() {
	controllers.ShutdownAll(r.scaledObjects, r.gateways, r.gatewayClasses)
}

func (r *clusterReadiness) ScalerReady(policy *clientnetworking.ServiceActivationPolicy) bool {
	ref := policy.Spec.GetAutoscalerRef()
	if ref == nil ||
		!strings.EqualFold(ref.GetGroup(), scaledObjectGVR.Group) ||
		!strings.EqualFold(ref.GetKind(), "ScaledObject") {
		return false
	}
	object := r.scaledObjects.Get(ref.GetName(), policy.GetNamespace())
	return scaledObjectReady(object)
}

func scaledObjectReady(object controllers.Object) bool {
	scaledObject, ok := object.(*unstructured.Unstructured)
	if !ok || scaledObject == nil {
		return false
	}
	conditions, found, err := unstructured.NestedSlice(scaledObject.Object, "status", "conditions")
	if err != nil || !found {
		return false
	}
	for _, item := range conditions {
		condition, ok := item.(map[string]any)
		if ok && condition["type"] == "Ready" && condition["status"] == conditionTrue {
			return true
		}
	}
	return false
}

func (r *clusterReadiness) ActivatorReady(policy *clientnetworking.ServiceActivationPolicy) bool {
	for _, gateway := range r.gateways.List(policy.GetNamespace(), klabels.Everything()) {
		class := r.gatewayClasses.Get(string(gateway.Spec.GatewayClassName), "")
		if class == nil || string(class.Spec.ControllerName) != features.ManagedGatewayController {
			continue
		}
		if gatewayProgrammed(gateway) {
			return true
		}
	}
	return false
}

func gatewayProgrammed(gateway *gatewayv1.Gateway) bool {
	for _, condition := range gateway.Status.Conditions {
		if condition.Type == string(gatewayv1.GatewayConditionProgrammed) &&
			condition.Status == metav1.ConditionTrue &&
			condition.ObservedGeneration == gateway.Generation {
			return true
		}
	}
	return false
}
