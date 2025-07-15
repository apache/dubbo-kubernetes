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

package dataplane

import (
	"context"
	"fmt"
	"maps"
	"time"

	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	config_core "github.com/apache/dubbo-kubernetes/pkg/config/mode"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_manager "github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"

	"github.com/pkg/errors"
	kube_apps "k8s.io/api/apps/v1"
	kube_core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	kube_ctrl "sigs.k8s.io/controller-runtime"
)



type dataplaneManager struct {
	core_manager.ResourceManager
	store      core_store.ResourceStore
	zone       string
	manager    kube_ctrl.Manager
	deployMode config_core.DeployMode
	clientset  *kubernetes.Clientset
}

func NewDataplaneManager(store core_store.ResourceStore, zone string, manager kube_ctrl.Manager, mode config_core.DeployMode) (core_manager.ResourceManager, error) {
	if mode == config_core.KubernetesMode || mode == config_core.HalfHostMode {
		clientset, err := kubernetes.NewForConfig(manager.GetConfig())
		if err != nil {
			return nil, fmt.Errorf("come up with error when initializing kubernetes.clientset: %w", err)
		}
		return &dataplaneManager{
			ResourceManager: core_manager.NewResourceManager(store),
			store:           store,
			zone:            zone,
			manager:         manager,
			deployMode:      mode,
			clientset:       clientset,
		}, nil
	}

	return &dataplaneManager{
		ResourceManager: core_manager.NewResourceManager(store),
		store:           store,
		zone:            zone,
		manager:         manager,
		deployMode:      mode,
		clientset:       nil,
	}, nil
}

func (m *dataplaneManager) Create(ctx context.Context, r core_model.Resource, fs ...core_store.CreateOptionsFunc) error {
	return m.store.Create(ctx, r, append(fs, core_store.CreatedAt(core.Now()))...)
}

func (m *dataplaneManager) Update(ctx context.Context, r core_model.Resource, fs ...core_store.UpdateOptionsFunc) error {
	return m.ResourceManager.Update(ctx, r, fs...)
}

func (m *dataplaneManager) Get(ctx context.Context, r core_model.Resource, opts ...core_store.GetOptionsFunc) error {
	dataplane, err := m.dataplane(r)
	if err != nil {
		return err
	}

	if err := m.store.Get(ctx, dataplane, opts...); err != nil {
		return err
	}
	m.setInboundsClusterTag(dataplane)
	m.setHealth(dataplane)
	if m.deployMode != config_core.UniversalMode {
		if m.deployMode == config_core.HalfHostMode {
			if err = m.mergeK8sPodMeta(ctx, dataplane); err != nil {
				return err
			}
		}
		m.setExtensions(ctx, dataplane)
	}
	return nil
}

func (m *dataplaneManager) List(ctx context.Context, r core_model.ResourceList, opts ...core_store.ListOptionsFunc) error {
	dataplanes, err := m.dataplanes(r)
	if err != nil {
		return err
	}
	if err := m.store.List(ctx, dataplanes, opts...); err != nil {
		return err
	}
	for _, item := range dataplanes.Items {
		m.setHealth(item)
		m.setInboundsClusterTag(item)
		if m.deployMode != config_core.UniversalMode {
			if m.deployMode == config_core.HalfHostMode {
				if err = m.mergeK8sPodMeta(ctx, item); err != nil {
					return err
				}
			}
			m.setExtensions(ctx, item)
		}
	}
	return nil
}

// replaceMeta is a ResourceMeta implementation that replace the following name extenions when set any of them:
// - "k8s.dubbo.io/namespace": replace with the field "namespace"
// - "k8s.dubbo.io/name": replace with the field "name"
// And replace the creation time when set the field "creationTime"
type replaceMeta struct {
	name         string
	namespace    string
	creationTime time.Time
	core_model.ResourceMeta
}

var _ core_model.ResourceMeta = &replaceMeta{}

func (m *replaceMeta) GetNameExtensions() core_model.ResourceNameExtensions {
	extensions := m.ResourceMeta.GetNameExtensions()
	if m.name == "" && m.namespace == "" {
		return extensions
	}
	if extensions == nil {
		return map[string]string{
			core_model.K8sNamespaceComponent: m.namespace,
			core_model.K8sNameComponent:      m.name,
		}
	} else {
		extensions = maps.Clone(extensions)
		extensions[core_model.K8sNamespaceComponent] = m.namespace
		extensions[core_model.K8sNameComponent] = m.name
		return extensions
	}
}

func (m *replaceMeta) GetCreationTime() time.Time {
	if m.creationTime.IsZero() {
		return m.ResourceMeta.GetCreationTime()
	} else {
		return m.creationTime
	}
}

// mergeK8sPodMeta merge k8s pod meta to dataplane
// so we can use the dataplane to merge extensions from pod
func (m *dataplaneManager) mergeK8sPodMeta(ctx context.Context, dp *core_mesh.DataplaneResource) error {
	if m.manager == nil {
		return nil
	}
	// get the pod related to the dataplane (ip)

	ip := dp.GetIP()
	pods, err := m.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: "status.podIP=" + ip,
	})
	if err != nil || len(pods.Items) == 0 {
		return err
	}
	pod := pods.Items[0]
	rm := &replaceMeta{
		name:         pod.Name,
		namespace:    pod.Namespace,
		ResourceMeta: dp.GetMeta(),
	}
	dp.SetMeta(rm)

	return nil
}

func (m *dataplaneManager) dataplane(resource core_model.Resource) (*core_mesh.DataplaneResource, error) {
	dp, ok := resource.(*core_mesh.DataplaneResource)
	if !ok {
		return nil, errors.Errorf("invalid resource type: expected=%T, got=%T", (*core_mesh.DataplaneResource)(nil), resource)
	}
	return dp, nil
}

func (m *dataplaneManager) dataplanes(resources core_model.ResourceList) (*core_mesh.DataplaneResourceList, error) {
	dp, ok := resources.(*core_mesh.DataplaneResourceList)
	if !ok {
		return nil, errors.Errorf("invalid resource type: expected=%T, got=%T", (*core_mesh.DataplaneResourceList)(nil), resources)
	}
	return dp, nil
}

func (m *dataplaneManager) setInboundsClusterTag(dp *core_mesh.DataplaneResource) {
	if m.zone == "" || dp.Spec.Networking == nil {
		return
	}

	for _, inbound := range dp.Spec.Networking.Inbound {
		if inbound.Tags == nil {
			inbound.Tags = make(map[string]string)
		}
		inbound.Tags[mesh_proto.ZoneTag] = m.zone
	}
}

func (m *dataplaneManager) setHealth(dp *core_mesh.DataplaneResource) {
	if m.zone == "" || dp.Spec.Networking == nil {
		return
	}

	for _, inbound := range dp.Spec.Networking.Inbound {
		if inbound.ServiceProbe != nil {
			inbound.State = mesh_proto.Dataplane_Networking_Inbound_NotReady
			// write health for backwards compatibility with Dubbo
			inbound.Health = &mesh_proto.Dataplane_Networking_Inbound_Health{
				Ready: false,
			}
		}
	}
}

func (m *dataplaneManager) setExtensions(ctx context.Context, dp *core_mesh.DataplaneResource) {
	if m.zone == "" || dp.Spec.Networking == nil {
		return
	}

	client := m.manager.GetClient()
	namespacedName := types.NamespacedName{
		Namespace: dp.GetMeta().GetNameExtensions()[core_model.K8sNamespaceComponent],
		Name:      dp.GetMeta().GetNameExtensions()[core_model.K8sNameComponent],
	}

	pod := &kube_core.Pod{}
	err := client.Get(ctx, namespacedName, pod)
	if err != nil {
		return
	}

	extensions := make(map[string]string)
	extensions[ExtensionsImageKey] = pod.Spec.Containers[0].Image

	podPhase := pod.Status.Phase
	extensions[ExtensionsPodPhaseKey] = string(podPhase)

	podStatusConditions := pod.Status.Conditions
	for _, condition := range podStatusConditions {
		if condition.Status != kube_core.ConditionTrue {
			extensions[ExtensionsPodStatusKey] = condition.Reason
			break
		}
	}

	containerState := pod.Status.ContainerStatuses[0].State
	if containerState.Waiting != nil {
		extensions[ExtensionsContainerStatusReasonKey] = containerState.Waiting.Reason
	} else if containerState.Terminated != nil {
		extensions[ExtensionsContainerStatusReasonKey] = containerState.Terminated.Reason
	}

	// get workload
	replicaSet := &kube_apps.ReplicaSet{}
	replicaSetNamespacedName := types.NamespacedName{Namespace: pod.Namespace, Name: pod.ObjectMeta.OwnerReferences[0].Name}
	if err = client.Get(ctx, replicaSetNamespacedName, replicaSet); err != nil {
		return
	}
	if pod.ObjectMeta.OwnerReferences[0].Name != "" {
		extensions[ExtensionsWorkLoadKey] = pod.ObjectMeta.OwnerReferences[0].Name
	} else if replicaSet.ObjectMeta.OwnerReferences[0].Name != "" {
		extensions[ExtensionsWorkLoadKey] = replicaSet.ObjectMeta.OwnerReferences[0].Name
	}

	// replace the creation time when the creation time is zero
	if dp.GetMeta().GetCreationTime().IsZero() {
		rm := &replaceMeta{
			ResourceMeta: dp.GetMeta(),
			creationTime: pod.CreationTimestamp.Time,
		}
		dp.SetMeta(rm)
	}

	// get NodeName
	extensions[ExtensionsNodeNameKey] = pod.Spec.NodeName
	for key, val := range extensions {
		dp.Spec.Extensions[key] = val
	}
}
