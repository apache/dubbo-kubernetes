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

package controller

import (
	"fmt"
	"sync"

	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kclient"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/hashicorp/go-multierror"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	mcs "sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
)

var (
	endpointSliceRequirement = labelRequirement(mcs.LabelServiceName, selection.DoesNotExist, nil)
	endpointSliceSelector    = klabels.NewSelector().Add(*endpointSliceRequirement)
)

type endpointSliceController struct {
	endpointCache *endpointSliceCache
	slices        kclient.Client[*v1.EndpointSlice]
	c             *Controller
}

type endpointSliceCache struct {
	mu                         sync.RWMutex
	endpointsByServiceAndSlice map[host.Name]map[string][]*model.DubboEndpoint
}

func newEndpointSliceCache() *endpointSliceCache {
	out := &endpointSliceCache{
		endpointsByServiceAndSlice: make(map[host.Name]map[string][]*model.DubboEndpoint),
	}
	return out
}

func newEndpointSliceController(c *Controller) *endpointSliceController {
	slices := kclient.NewFiltered[*v1.EndpointSlice](c.client, kclient.Filter{ObjectFilter: c.client.ObjectFilter()})
	out := &endpointSliceController{
		c:             c,
		slices:        slices,
		endpointCache: newEndpointSliceCache(),
	}
	registerHandlers[*v1.EndpointSlice](c, slices, "EndpointSlice", out.onEvent, nil)
	return out
}

func (esc *endpointSliceController) onEvent(_, ep *v1.EndpointSlice, event model.Event) error {
	esc.onEventInternal(nil, ep, event)
	return nil
}

func (esc *endpointSliceController) onEventInternal(_, ep *v1.EndpointSlice, event model.Event) {
	esLabels := ep.GetLabels()
	if !endpointSliceSelector.Matches(klabels.Set(esLabels)) {
		return
	}
	if event == model.EventDelete {
		esc.deleteEndpointSlice(ep)
	} else {
		esc.updateEndpointSlice(ep)
	}

	// Now check if we need to do a full push for the service.
	// If the service is headless, we need to do a full push if service exposes TCP ports
	// to create IP based listeners. For pure HTTP headless services, we only need to push NDS.
	name := serviceNameForEndpointSlice(esLabels)
	namespace := ep.GetNamespace()
	svc := esc.c.services.Get(name, namespace)
	if svc != nil && !serviceNeedsPush(svc) {
		return
	}

	namespacedName := getServiceNamespacedName(ep)
	hostnames := esc.c.hostNamesForNamespacedName(namespacedName)
	esc.pushEDS(hostnames, namespacedName.Namespace)

	if svc == nil || svc.Spec.ClusterIP != corev1.ClusterIPNone || svc.Spec.Type == corev1.ServiceTypeExternalName {
		return
	}

	configsUpdated := sets.New[model.ConfigKey]()
	supportsOnlyHTTP := true
	for _, modelSvc := range esc.c.servicesForNamespacedName(config.NamespacedName(svc)) {
		for _, p := range modelSvc.Ports {
			if !p.Protocol.IsHTTP() {
				supportsOnlyHTTP = false
				break
			}
		}
		if supportsOnlyHTTP {
			configsUpdated.Insert(model.ConfigKey{Kind: kind.ServiceEntry, Name: modelSvc.Hostname.String(), Namespace: svc.Namespace})
		}
	}

	if len(configsUpdated) > 0 {
		esc.c.opts.XDSUpdater.ConfigUpdate(&model.PushRequest{
			Full:           true,
			ConfigsUpdated: configsUpdated,
			Reason:         model.NewReasonStats(model.HeadlessEndpointUpdate),
		})
	}
}

func (esc *endpointSliceController) deleteEndpointSlice(slice *v1.EndpointSlice) {
	key := config.NamespacedName(slice)
	for _, e := range slice.Endpoints {
		for _, a := range e.Addresses {
			esc.c.pods.endpointDeleted(key, a)
		}
	}

	esc.endpointCache.mu.Lock()
	defer esc.endpointCache.mu.Unlock()
	for _, hostName := range esc.c.hostNamesForNamespacedName(getServiceNamespacedName(slice)) {
		// endpointSlice cache update
		if esc.endpointCache.has(hostName) {
			esc.endpointCache.delete(hostName, slice.Name)
		}
	}
}

func (e *endpointSliceCache) has(hostname host.Name) bool {
	_, found := e.endpointsByServiceAndSlice[hostname]
	return found
}

func (e *endpointSliceCache) delete(hostname host.Name, slice string) {
	delete(e.endpointsByServiceAndSlice[hostname], slice)
	if len(e.endpointsByServiceAndSlice[hostname]) == 0 {
		delete(e.endpointsByServiceAndSlice, hostname)
	}
}

func (esc *endpointSliceController) updateEndpointSlice(slice *v1.EndpointSlice) {
	for _, hostname := range esc.c.hostNamesForNamespacedName(getServiceNamespacedName(slice)) {
		esc.updateEndpointCacheForSlice(hostname, slice)
	}
}

func (esc *endpointSliceController) initializeNamespace(ns string, filtered bool) error {
	var err *multierror.Error
	var endpoints []*v1.EndpointSlice
	if filtered {
		endpoints = esc.slices.List(ns, klabels.Everything())
	} else {
		endpoints = esc.slices.ListUnfiltered(ns, klabels.Everything())
	}
	for _, s := range endpoints {
		err = multierror.Append(err, esc.onEvent(nil, s, model.EventAdd))
	}
	return err.ErrorOrNil()
}

func (esc *endpointSliceController) updateEndpointCacheForSlice(hostName host.Name, epSlice *v1.EndpointSlice) {
	var endpoints []*model.DubboEndpoint
	if epSlice.AddressType == v1.AddressTypeFQDN {
		return
	}
	svc := esc.c.GetService(hostName)
	svcNamespacedName := getServiceNamespacedName(epSlice)
	// This is not a endpointslice for service, ignore
	if svcNamespacedName.Name == "" {
		return
	}

	for _, e := range epSlice.Endpoints {
		// Draining tracking is only enabled if persistent sessions is enabled.
		// If we start using them for other features, this can be adjusted.
		healthStatus := endpointHealthStatus(svc, e)

		if len(e.Addresses) > 0 {
			ready := "nil"
			terminating := "nil"
			if e.Conditions.Ready != nil {
				ready = fmt.Sprintf("%v", *e.Conditions.Ready)
			}
			if e.Conditions.Terminating != nil {
				terminating = fmt.Sprintf("%v", *e.Conditions.Terminating)
			}
			log.Debugf("endpointHealthStatus: address=%s, Ready=%s, Terminating=%s, HealthStatus=%v, svc=%v",
				e.Addresses[0], ready, terminating, healthStatus, svc != nil)
		}

		for _, a := range e.Addresses {
			pod, expectedPod := getPod(esc.c, a, &metav1.ObjectMeta{Name: epSlice.Name, Namespace: epSlice.Namespace}, e.TargetRef, hostName)
			if pod == nil && expectedPod {
				continue
			}

			var overrideAddresses []string
			builder := esc.c.NewEndpointBuilder(pod)
			// EDS and ServiceEntry use name for service port - ADS will need to map to numbers.
			// Always use Service.Port.Name as source of truth for ServicePortName
			// - Use EndpointSlice.Port.Port (service port number) as endpointPort
			// - Always resolve portName from Service by matching port number (Service is source of truth)
			// - This ensures ep.ServicePortName matches svcPort.Name in BuildClusterLoadAssignment
			kubeSvc := esc.c.services.Get(svcNamespacedName.Name, svcNamespacedName.Namespace)
			for _, port := range epSlice.Ports {
				var epSlicePortNum int32
				if port.Port != nil {
					epSlicePortNum = *port.Port
				}
				var epSlicePortName string
				if port.Name != nil {
					epSlicePortName = *port.Name
				}

				var servicePortNum int32
				var targetPortNum int32
				var portName string

				// EndpointSlice.Port.Port might be targetPort or service port
				// We need to find the matching ServicePort and resolve:
				// 1. servicePortNum (Service.Port) - used for matching in BuildClusterLoadAssignment
				// 2. targetPortNum (Service.TargetPort) - used as EndpointPort
				// 3. portName (Service.Port.Name) - used as ServicePortName for filtering
				if kubeSvc != nil {
					matched := false
					for _, kubePort := range kubeSvc.Spec.Ports {
						// Try matching by port number (service port or targetPort)
						portMatches := int32(kubePort.Port) == epSlicePortNum

						// Also try matching by targetPort
						var kubeTargetPort int32
						if kubePort.TargetPort.Type == intstr.Int {
							kubeTargetPort = kubePort.TargetPort.IntVal
						} else if kubePort.TargetPort.Type == intstr.String && pod != nil {
							// Resolve targetPort name from Pod
							for _, container := range pod.Spec.Containers {
								for _, containerPort := range container.Ports {
									if containerPort.Name == kubePort.TargetPort.StrVal {
										kubeTargetPort = containerPort.ContainerPort
										break
									}
								}
								if kubeTargetPort != 0 {
									break
								}
							}
						}
						targetPortMatches := kubeTargetPort == epSlicePortNum

						// Match by port name if available
						nameMatches := epSlicePortName == "" || kubePort.Name == epSlicePortName

						if (portMatches || targetPortMatches) && (epSlicePortName == "" || nameMatches) {
							// Found matching ServicePort
							servicePortNum = int32(kubePort.Port)
							portName = kubePort.Name

							// Resolve targetPortNum
							if kubePort.TargetPort.Type == intstr.Int {
								targetPortNum = kubePort.TargetPort.IntVal
							} else if kubePort.TargetPort.Type == intstr.String && pod != nil {
								// Resolve targetPort name from Pod container ports
								for _, container := range pod.Spec.Containers {
									for _, containerPort := range container.Ports {
										if containerPort.Name == kubePort.TargetPort.StrVal {
											targetPortNum = containerPort.ContainerPort
											break
										}
									}
									if targetPortNum != 0 {
										break
									}
								}
							} else {
								// If targetPort is string but pod not found, use service port as fallback
								targetPortNum = servicePortNum
							}

							// If targetPortNum is still 0, use service port
							if targetPortNum == 0 {
								targetPortNum = servicePortNum
							}

							matched = true
							log.Debugf("updateEndpointCacheForSlice: matched ServicePort (servicePort=%d, targetPort=%d, portName='%s') for EndpointSlice.Port (portNum=%d, portName='%s')",
								servicePortNum, targetPortNum, portName, epSlicePortNum, epSlicePortName)
							break
						}
					}

					if !matched {
						// If we can't match by Service, try to find ServicePort by port number only
						// This handles cases where EndpointSlice.Port.Name doesn't match but port number does
						for _, kubePort := range kubeSvc.Spec.Ports {
							if int32(kubePort.Port) == epSlicePortNum {
								// Found by port number, use Service.Port.Name
								servicePortNum = int32(kubePort.Port)
								portName = kubePort.Name
								// Resolve targetPortNum
								if kubePort.TargetPort.Type == intstr.Int {
									targetPortNum = kubePort.TargetPort.IntVal
								} else if kubePort.TargetPort.Type == intstr.String && pod != nil {
									for _, container := range pod.Spec.Containers {
										for _, containerPort := range container.Ports {
											if containerPort.Name == kubePort.TargetPort.StrVal {
												targetPortNum = containerPort.ContainerPort
												break
											}
										}
										if targetPortNum != 0 {
											break
										}
									}
								}
								if targetPortNum == 0 {
									targetPortNum = servicePortNum
								}
								matched = true
								log.Infof("updateEndpointCacheForSlice: matched ServicePort by port number only (servicePort=%d, targetPort=%d, portName='%s') for EndpointSlice.Port (portNum=%d, portName='%s')",
									servicePortNum, targetPortNum, portName, epSlicePortNum, epSlicePortName)
								break
							}
						}

						if !matched {
							log.Warnf("updateEndpointCacheForSlice: failed to match EndpointSlice.Port (portNum=%d, portName='%s') with Service %s, using EndpointSlice values (WARNING: portName may not match Service.Port.Name)",
								epSlicePortNum, epSlicePortName, svcNamespacedName.Name)
							// Fallback: use EndpointSlice values
							// This should rarely happen, but if it does, portName may not match Service.Port.Name
							// which will cause endpoints to be filtered in BuildClusterLoadAssignment
							servicePortNum = epSlicePortNum
							targetPortNum = epSlicePortNum
							if epSlicePortName != "" {
								portName = epSlicePortName
							} else {
								// If EndpointSlice.Port.Name is also empty, try to get port name from Service by port number
								for _, kubePort := range kubeSvc.Spec.Ports {
									if int32(kubePort.Port) == epSlicePortNum {
										portName = kubePort.Name
										log.Debugf("updateEndpointCacheForSlice: resolved portName='%s' from Service by port number %d", portName, epSlicePortNum)
										break
									}
								}
							}
						}
					}
				} else {
					// Service not found, use EndpointSlice values
					servicePortNum = epSlicePortNum
					targetPortNum = epSlicePortNum
					if epSlicePortName != "" {
						portName = epSlicePortName
					}
					log.Debugf("updateEndpointCacheForSlice: Service not found for %s, using EndpointSlice values (portNum=%d, portName='%s')",
						svcNamespacedName.Name, epSlicePortNum, epSlicePortName)
				}

				log.Debugf("updateEndpointCacheForSlice: creating endpoint for service %s (address=%s, servicePortNum=%d, targetPortNum=%d, portName='%s', hostname=%s, kubeSvc=%v)",
					svcNamespacedName.Name, a, servicePortNum, targetPortNum, portName, hostName, kubeSvc != nil)

				// - EndpointSlice.Port.Port should be the Service Port (not targetPort)
				// - But IstioEndpoint.EndpointPort should be the targetPort (container port)
				// - ServicePortName should match Service.Port.Name for filtering in BuildClusterLoadAssignment
				//
				// We use targetPortNum as EndpointPort because that's what the container actually listens on.
				// The servicePortNum is used for matching in BuildClusterLoadAssignment via portName.
				//
				// svc.SupportsUnhealthyEndpoints() returns true if Service has publishNotReadyAddresses=true
				// This allows endpoints with Ready=false to be included in EDS, which is useful for services
				// that need to receive traffic even before they are fully ready (e.g., during startup).
				// Check if svc is nil before calling SupportsUnhealthyEndpoints
				var supportsUnhealthy bool
				if svc != nil {
					supportsUnhealthy = svc.SupportsUnhealthyEndpoints()
				} else {
					// If service is nil, default to false (don't support unhealthy endpoints)
					supportsUnhealthy = false
				}
				dubboEndpoint := builder.buildDubboEndpoint(a, targetPortNum, portName, nil, healthStatus, supportsUnhealthy)

				// Log if endpoint is unhealthy and service doesn't support it
				if healthStatus == model.UnHealthy && !supportsUnhealthy {
					if svc != nil {
						log.Debugf("updateEndpointCacheForSlice: endpoint %s is unhealthy (HealthStatus=%v) but service %s does not support unhealthy endpoints (PublishNotReadyAddresses=%v). Endpoint will be filtered in EDS.",
							a, healthStatus, svcNamespacedName.Name, svc.Attributes.PublishNotReadyAddresses)
					} else {
						log.Debugf("updateEndpointCacheForSlice: endpoint %s is unhealthy (HealthStatus=%v) but service %s is nil. Endpoint will be filtered in EDS.",
							a, healthStatus, svcNamespacedName.Name)
					}
				}

				// Verify the endpoint was created with correct ServicePortName
				if dubboEndpoint != nil {
					log.Debugf("updateEndpointCacheForSlice: created endpoint with ServicePortName='%s', EndpointPort=%d, address=%s",
						dubboEndpoint.ServicePortName, dubboEndpoint.EndpointPort, dubboEndpoint.FirstAddressOrNil())
				} else {
					log.Debugf("updateEndpointCacheForSlice: buildDubboEndpoint returned nil for address=%s, targetPortNum=%d, portName='%s'",
						a, targetPortNum, portName)
				}
				if len(overrideAddresses) > 1 {
					dubboEndpoint.Addresses = overrideAddresses
				}
				endpoints = append(endpoints, dubboEndpoint)
			}
		}
	}
	esc.endpointCache.Update(hostName, epSlice.Name, endpoints)
}

func (e *endpointSliceCache) Update(hostname host.Name, slice string, endpoints []*model.DubboEndpoint) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.update(hostname, slice, endpoints)
}

func (e *endpointSliceCache) update(hostname host.Name, slice string, endpoints []*model.DubboEndpoint) {
	if len(endpoints) == 0 {
		delete(e.endpointsByServiceAndSlice[hostname], slice)
	}
	if _, f := e.endpointsByServiceAndSlice[hostname]; !f {
		e.endpointsByServiceAndSlice[hostname] = make(map[string][]*model.DubboEndpoint)
	}
	// We will always overwrite. A conflict here means an endpoint is transitioning
	// from one slice to another See
	// https://github.com/kubernetes/website/blob/master/content/en/docs/concepts/services-networking/endpoint-slices.md#duplicate-endpoints
	// In this case, we can always assume and update is fresh, although older slices
	// we have not gotten updates may be stale; therefore we always take the new
	// update.
	e.endpointsByServiceAndSlice[hostname][slice] = endpoints
}

func endpointHealthStatus(svc *model.Service, e v1.Endpoint) model.HealthStatus {
	// Correct health status logic
	// 1. If Ready is nil or true, endpoint is healthy
	// 2. If Ready is false, check if it's terminating
	// 3. If terminating, mark as Terminating
	// 4. Otherwise, mark as UnHealthy

	// Check Ready condition
	if e.Conditions.Ready == nil || *e.Conditions.Ready {
		return model.Healthy
	}

	// Ready is false, check if it's terminating
	// Terminating should be checked only if it's not nil AND true
	// If Terminating is nil, it means the endpoint is not terminating (it's just not ready)
	if e.Conditions.Terminating != nil && *e.Conditions.Terminating {
		return model.Terminating
	}

	// Ready is false and not terminating, mark as unhealthy
	return model.UnHealthy
}

func serviceNameForEndpointSlice(labels map[string]string) string {
	return labels[v1.LabelServiceName]
}

func getPod(c *Controller, ip string, ep *metav1.ObjectMeta, targetRef *corev1.ObjectReference, host host.Name) (*corev1.Pod, bool) {
	var expectPod bool
	pod := c.getPod(ip, ep.Namespace, targetRef)
	if targetRef != nil && targetRef.Kind == kind.Pod.String() {
		expectPod = true
		if pod == nil {
			c.registerEndpointResync(ep, ip, host)
		}
	}

	return pod, expectPod
}

func (c *Controller) registerEndpointResync(ep *metav1.ObjectMeta, ip string, host host.Name) {
	// Tell pod cache we want to queue the endpoint event when this pod arrives.
	c.pods.queueEndpointEventOnPodArrival(config.NamespacedName(ep), ip)
}

func (c *Controller) getPod(ip string, namespace string, targetRef *corev1.ObjectReference) *corev1.Pod {
	if targetRef != nil && targetRef.Kind == kind.Pod.String() {
		key := types.NamespacedName{Name: targetRef.Name, Namespace: targetRef.Namespace}
		pod := c.pods.getPodByKey(key)
		return pod
	}
	// This means the endpoint is manually controlled
	// We will want to lookup a pod to find metadata like service account, labels, etc. But for hostNetwork, we just get a raw IP,
	// and the IP may be shared by many pods. Best we can do is guess.
	pods := c.pods.getPodsByIP(ip)
	for _, p := range pods {
		if p.Namespace == namespace {
			// Might not be right, but best we can do.
			return p
		}
	}
	return nil
}

func getServiceNamespacedName(slice *v1.EndpointSlice) types.NamespacedName {
	return types.NamespacedName{
		Namespace: slice.GetNamespace(),
		Name:      serviceNameForEndpointSlice(slice.GetLabels()),
	}
}

func serviceNeedsPush(svc *corev1.Service) bool {
	// In proxyless mesh, all services need to be pushed
	return true
}

func (esc *endpointSliceController) pushEDS(hostnames []host.Name, namespace string) {
	shard := model.ShardKeyFromRegistry(esc.c)
	esc.endpointCache.mu.Lock()
	defer esc.endpointCache.mu.Unlock()

	for _, hostname := range hostnames {
		endpoints := esc.endpointCache.get(hostname)
		log.Debugf("pushEDS: registering %d endpoints for service %s in namespace %s (shard=%v)",
			len(endpoints), string(hostname), namespace, shard)
		if len(endpoints) > 0 {
			// Log endpoint details for first few endpoints
			for i, ep := range endpoints {
				if i < 3 {
					log.Debugf("pushEDS: endpoint[%d] address=%s, port=%d, ServicePortName='%s', HealthStatus=%v",
						i, ep.FirstAddressOrNil(), ep.EndpointPort, ep.ServicePortName, ep.HealthStatus)
				}
			}
		}
		esc.c.opts.XDSUpdater.EDSUpdate(shard, string(hostname), namespace, endpoints)
	}
}

type endpointKey struct {
	ip   string
	port string
}

func (e *endpointSliceCache) get(hostname host.Name) []*model.DubboEndpoint {
	var endpoints []*model.DubboEndpoint
	found := sets.New[endpointKey]()
	for _, eps := range e.endpointsByServiceAndSlice[hostname] {
		for _, ep := range eps {
			key := endpointKey{ep.FirstAddressOrNil(), ep.ServicePortName}
			if found.InsertContains(key) {
				// This a duplicate. Update() already handles conflict resolution, so we don't
				// need to pick the "right" one here.
				continue
			}
			endpoints = append(endpoints, ep)
		}
	}
	return endpoints
}

func (esc *endpointSliceController) podArrived(name, ns string) error {
	ep := esc.slices.Get(name, ns)
	if ep == nil {
		return nil
	}
	return esc.onEvent(nil, ep, model.EventAdd)
}

func (esc *endpointSliceController) buildDubboEndpointsWithService(name, namespace string, hostName host.Name, updateCache bool) []*model.DubboEndpoint {
	esLabelSelector := endpointSliceSelectorForService(name)
	slices := esc.slices.List(namespace, esLabelSelector)
	if len(slices) == 0 {
		return nil
	}

	if updateCache {
		// A cache update was requested. Rebuild the endpoints for these slices.
		for _, slice := range slices {
			esc.updateEndpointCacheForSlice(hostName, slice)
		}
	}

	return esc.endpointCache.Get(hostName)
}

func (e *endpointSliceCache) Get(hostname host.Name) []*model.DubboEndpoint {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.get(hostname)
}

func endpointSliceSelectorForService(name string) klabels.Selector {
	return klabels.Set(map[string]string{
		v1.LabelServiceName: name,
	}).AsSelectorPreValidated().Add(*endpointSliceRequirement)
}
