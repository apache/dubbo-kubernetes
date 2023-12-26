package selector

import (
	"github.com/apache/dubbo-kubernetes/pkg/admin/constant"
	"k8s.io/apimachinery/pkg/labels"
)

type Selector interface {
	AsLabelsSelector() labels.Selector

	ApplicationOption() (string, bool)
}
type ApplicationSelector struct {
	Name string
}

func (s *ApplicationSelector) AsLabelsSelector() labels.Selector {
	selector := labels.Set{
		constant.ApplicationLabel: s.Name,
	}
	return selector.AsSelector()
}

func (s *ApplicationSelector) ApplicationOption() (string, bool) {
	return s.Name, true
}
