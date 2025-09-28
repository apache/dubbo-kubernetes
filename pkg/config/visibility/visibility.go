package visibility

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/config/labels"
)

type Instance string

const (
	Private Instance = "."
	Public  Instance = "*"
	None    Instance = "~"
)

func (v Instance) Validate() (errs error) {
	switch v {
	case Private, Public:
		return nil
	case None:
		return fmt.Errorf("exportTo ~ (none) is not allowed for Istio configuration objects")
	default:
		if !labels.IsDNS1123Label(string(v)) {
			return fmt.Errorf("only .,*, or a valid DNS 1123 label is allowed as exportTo entry")
		}
	}
	return nil
}
