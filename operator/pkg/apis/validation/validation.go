package validation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/apache/dubbo-kubernetes/operator/pkg/apis"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util"
	"github.com/apache/dubbo-kubernetes/operator/pkg/values"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"sigs.k8s.io/yaml"
)

type Warnings = util.Errors

func ParseAndValidateDubboOperator(dopm values.Map, client kube.CLIClient) (Warnings, util.Errors) {
	iop := &apis.DubboOperator{}
	dec := json.NewDecoder(bytes.NewBufferString(dopm.JSON()))
	dec.DisallowUnknownFields()
	if err := dec.Decode(iop); err != nil {
		return nil, util.NewErrs(fmt.Errorf("could not unmarshal: %v", err))
	}
	var warnings Warnings
	var errors util.Errors

	vw, ve := validateValues(iop)
	warnings = util.AppendErrs(warnings, vw)
	errors = util.AppendErrs(errors, ve)
	errors = util.AppendErr(errors, validateComponentNames(iop.Spec.Components))
	return warnings, errors
}

type FeatureValidator func(*apis.Values, apis.DubboOperatorSpec) (Warnings, util.Errors)

func validateFeatures(values *apis.Values, spec apis.DubboOperatorSpec) (Warnings, util.Errors) {
	validators := []FeatureValidator{}
	var warnings Warnings
	var errs util.Errors
	for _, validator := range validators {
		newWarnings, newErrs := validator(values, spec)
		errs = util.AppendErrs(errs, newErrs)
		warnings = append(warnings, newWarnings...)
	}
	return warnings, errs
}

func validateValues(raw *apis.DubboOperator) (Warnings, util.Errors) {
	values := &apis.Values{}
	if err := yaml.Unmarshal(raw.Spec.Values, values); err != nil {
		return nil, util.NewErrs(fmt.Errorf("could not unmarshal: %v", err))
	}
	warnings, errs := validateFeatures(values, raw.Spec)

	return warnings, errs
}

func validateComponentNames(components *apis.DubboComponentSpec) error {
	if components == nil {
		return nil
	}
	return nil
}
