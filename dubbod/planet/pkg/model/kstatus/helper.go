package kstatus

import (
	"github.com/apache/dubbo-kubernetes/pkg/slices"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	StatusTrue  = "True"
	StatusFalse = "False"
)

func UpdateConditionIfChanged(conditions []metav1.Condition, condition metav1.Condition) []metav1.Condition {
	ret := slices.Clone(conditions)
	existing := slices.FindFunc(ret, func(cond metav1.Condition) bool {
		return cond.Type == condition.Type
	})
	if existing == nil {
		ret = append(ret, condition)
		return ret
	}

	if existing.Status == condition.Status {
		if existing.Message == condition.Message &&
			existing.ObservedGeneration == condition.ObservedGeneration {
			return conditions
		}
		condition.LastTransitionTime = existing.LastTransitionTime
	}
	*existing = condition

	return ret
}











