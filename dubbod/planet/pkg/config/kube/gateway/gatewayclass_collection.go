package gateway

import (
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	gateway "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type GatewayClass struct {
	Name       string
	Controller gateway.GatewayController
}

func (g GatewayClass) ResourceName() string {
	return g.Name
}

func getKnownControllerNames() []string {
	names := make([]string, 0, len(classInfos))
	for k := range classInfos {
		names = append(names, string(k))
	}
	return names
}

func GatewayClassesCollection(
	gatewayClasses krt.Collection[*gateway.GatewayClass],
	opts krt.OptionsBuilder,
) (
	krt.StatusCollection[*gateway.GatewayClass, gatewayv1.GatewayClassStatus],
	krt.Collection[GatewayClass],
) {
	return krt.NewStatusCollection(gatewayClasses, func(ctx krt.HandlerContext, obj *gateway.GatewayClass) (*gatewayv1.GatewayClassStatus, *GatewayClass) {
		log.Infof("Processing GatewayClass %s with controller name %q", obj.Name, obj.Spec.ControllerName)
		// Log all known controller names for debugging
		for k := range classInfos {
			log.Infof("Known controller in classInfos: %q", string(k))
		}
		_, known := classInfos[obj.Spec.ControllerName]
		if !known {
			log.Warnf("GatewayClass %s has unknown controller name %q, skipping status update. Known controllers: %v", 
				obj.Name, obj.Spec.ControllerName, getKnownControllerNames())
			return nil, nil
		}
		log.Infof("GatewayClass %s has known controller name %q, updating status", obj.Name, obj.Spec.ControllerName)
		status := obj.Status.DeepCopy()
		status = GetClassStatus(status, obj.Generation)
		return status, &GatewayClass{
			Name:       obj.Name,
			Controller: obj.Spec.ControllerName,
		}
	}, opts.WithName("GatewayClasses")...)
}
