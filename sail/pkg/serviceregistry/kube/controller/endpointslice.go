package controller

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/kind"
	"github.com/apache/dubbo-kubernetes/pkg/config/visibility"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kclient"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	"istio.io/api/annotation"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/discovery/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	mcs "sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
	"strings"
)

var (
	endpointSliceRequirement = labelRequirement(mcs.LabelServiceName, selection.DoesNotExist, nil)
	endpointSliceSelector    = klabels.NewSelector().Add(*endpointSliceRequirement)
)

type endpointSliceController struct {
	slices kclient.Client[*v1.EndpointSlice]
	c      *Controller
}

func newEndpointSliceController(c *Controller) *endpointSliceController {
	slices := kclient.NewFiltered[*v1.EndpointSlice](c.client, kclient.Filter{ObjectFilter: c.client.ObjectFilter()})
	out := &endpointSliceController{
		c:      c,
		slices: slices,
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
	// Update internal endpoint cache no matter what kind of service, even headless service.
	// As for gateways, the cluster discovery type is `EDS` for headless service.
	// namespacedName := getServiceNamespacedName(ep)
	// log.Debugf("Handle EDS endpoint %s %s in namespace %s", namespacedName.Name, event, namespacedName.Namespace)
	// if event == model.EventDelete {
	// 	esc.deleteEndpointSlice(ep)
	// } else {
	// 	esc.updateEndpointSlice(ep)
	// }

	// Now check if we need to do a full push for the service.
	// If the service is headless, we need to do a full push if service exposes TCP ports
	// to create IP based listeners. For pure HTTP headless services, we only need to push NDS.
	name := serviceNameForEndpointSlice(esLabels)
	namespace := ep.GetNamespace()
	svc := esc.c.services.Get(name, namespace)
	if svc != nil && !serviceNeedsPush(svc) {
		return
	}

	// hostnames := esc.c.hostNamesForNamespacedName(namespacedName)
	// log.Debugf("triggering EDS push for %s in namespace %s", hostnames, namespacedName.Namespace)
	// Trigger EDS push for all hostnames.
	// esc.pushEDS(hostnames, namespacedName.Namespace)

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
			// pure HTTP headless services should not need a full push since they do not
			// require a Listener based on IP: https://github.com/istio/istio/issues/48207
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

func serviceNameForEndpointSlice(labels map[string]string) string {
	return labels[v1.LabelServiceName]
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
