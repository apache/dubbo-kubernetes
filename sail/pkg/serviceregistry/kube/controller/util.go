package controller

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/config/labels"
	v1 "k8s.io/api/core/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func labelRequirement(key string, op selection.Operator, vals []string, opts ...field.PathOption) *klabels.Requirement {
	out, err := klabels.NewRequirement(key, op, vals, opts...)
	if err != nil {
		panic(fmt.Sprintf("failed creating requirements for Service: %v", err))
	}
	return out
}

func getPodServices(allServices []*v1.Service, pod *v1.Pod) []*v1.Service {
	var services []*v1.Service
	for _, service := range allServices {
		if labels.Instance(service.Spec.Selector).Match(pod.Labels) {
			services = append(services, service)
		}
	}

	return services
}
