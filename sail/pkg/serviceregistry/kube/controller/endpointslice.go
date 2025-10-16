package controller

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/host"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/config/visibility"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kclient"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"github.com/hashicorp/go-multierror"
	"istio.io/api/annotation"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	mcs "sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
	"strings"
	"sync"
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
			configsUpdated.Insert(model.ConfigKey{Kind: kind.DNSName, Name: modelSvc.Hostname.String(), Namespace: svc.Namespace})
		} else {
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
		for _, a := range e.Addresses {
			pod, expectedPod := getPod(esc.c, a, &metav1.ObjectMeta{Name: epSlice.Name, Namespace: epSlice.Namespace}, e.TargetRef, hostName)
			if pod == nil && expectedPod {
				continue
			}

			var overrideAddresses []string
			builder := esc.c.NewEndpointBuilder(pod)
			// EDS and ServiceEntry use name for service port - ADS will need to map to numbers.
			for _, port := range epSlice.Ports {
				var portNum int32
				if port.Port != nil {
					portNum = *port.Port
				}
				var portName string
				if port.Name != nil {
					portName = *port.Name
				}

				dubboEndpoint := builder.buildDubboEndpoint(a, portNum, portName, nil, healthStatus, svc.SupportsUnhealthyEndpoints())
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
	if e.Conditions.Ready == nil || *e.Conditions.Ready {
		return model.Healthy
	}

	// If it is shutting down, mark it as terminating. This occurs regardless of whether it was previously healthy or not.
	if svc != nil &&
		(e.Conditions.Terminating == nil || *e.Conditions.Terminating) {
		return model.Terminating
	}

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
	if svc.Annotations[annotation.NetworkingExportTo.Name] != "" {
		namespaces := strings.Split(svc.Annotations[annotation.NetworkingExportTo.Name], ",")
		for _, ns := range namespaces {
			ns = strings.TrimSpace(ns)
			if ns == string(visibility.None) {
				return false
			}
		}
	}
	return true
}

func (esc *endpointSliceController) pushEDS(hostnames []host.Name, namespace string) {
	shard := model.ShardKeyFromRegistry(esc.c)
	esc.endpointCache.mu.Lock()
	defer esc.endpointCache.mu.Unlock()

	for _, hostname := range hostnames {
		endpoints := esc.endpointCache.get(hostname)
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
