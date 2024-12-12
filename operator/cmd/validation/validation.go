package validation

import (
	"bytes"
	"encoding/json"
	"github.com/apache/dubbo-kubernetes/operator/pkg/apis"
	"github.com/apache/dubbo-kubernetes/operator/pkg/values"
)

func ParseAndValidateDubboOperator(dopMap values.Map) (Warnings, util.Errors) {
	dop := &apis.DubboOperator{}
	dec := json.NewDecoder(bytes.NewBufferString(dopMap.JSON()))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dop); err != nil {
		return nil, util.NewErrs(fmt.Errorf("could not unmarshal: %v", err))
	}
	var warnings Warnings
	var errors util.Errors
	vw, ve := validateValues(dop)
	warnings = util.AppendErrs(warnings, vw)
	errors = util.AppendErrs(errors, ve)
	errors = util.AppendErr(errors, validateComponentNames(dop.Spec.Components))
	return warnings, errors
}

func validateValues(raw *apis.DubboOperator) (Warnings, util.Errors) {
	v := &apis.Values{}
	if err := yaml.Unmarshal(raw.Spec.Values, v); err != nil {
		return nil, util.NewErrs(fmt.Errorf("could not unmarshal: %v", err))
	}
	return nil, nil
}
