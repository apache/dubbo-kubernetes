package gateway

import (
	"github.com/apache/dubbo-kubernetes/dubbod/planet/pkg/model/kstatus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func GetClassStatus(existing *gatewayv1.GatewayClassStatus, gen int64) *gatewayv1.GatewayClassStatus {
	if existing == nil {
		existing = &gatewayv1.GatewayClassStatus{}
	}
	existing.Conditions = kstatus.UpdateConditionIfChanged(existing.Conditions, metav1.Condition{
		Type:               string(gatewayv1.GatewayClassConditionStatusAccepted),
		Status:             kstatus.StatusTrue,
		ObservedGeneration: gen,
		LastTransitionTime: metav1.Now(),
		Reason:             string(gatewayv1.GatewayClassConditionStatusAccepted),
		Message:            "Handled by Dubbo controller",
	})
	return existing
}











