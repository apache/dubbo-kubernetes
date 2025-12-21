package gateway

import (
	"github.com/apache/dubbo-kubernetes/pkg/util/ptr"
	k8sv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func GetStatus[I, IS any](spec I) IS {
	switch t := any(spec).(type) {
	case *k8sv1.HTTPRoute:
		return any(t.Status).(IS)
	case *k8sv1.Gateway:
		return any(t.Status).(IS)
	case *k8sv1.GatewayClass:
		return any(t.Status).(IS)
	default:
		log.Fatalf("unknown type %T", t)
		return ptr.Empty[IS]()
	}
}
