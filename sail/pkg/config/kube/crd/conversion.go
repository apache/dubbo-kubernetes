package crd

import (
	"bytes"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/resource"
	"io"
	kubeyaml "k8s.io/apimachinery/pkg/util/yaml"
	"reflect"
)

type ConversionFunc = func(s resource.Schema, js string) (config.Spec, error)

// TODO - add special cases for type-to-kind and kind-to-type
// conversions with initial-isms. Consider adding additional type
// information to the abstract model and/or elevating k8s
// representation to first-class type to avoid extra conversions.

func parseInputsImpl(inputs string, withValidate bool) ([]config.Config, []DubboKind, error) {
	var varr []config.Config
	var others []DubboKind
	reader := bytes.NewReader([]byte(inputs))
	empty := DubboKind{}

	// We store configs as a YaML stream; there may be more than one decoder.
	yamlDecoder := kubeyaml.NewYAMLOrJSONDecoder(reader, 512*1024)
	for {
		obj := DubboKind{}
		err := yamlDecoder.Decode(&obj)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("cannot parse proto message: %v", err)
		}
		if reflect.DeepEqual(obj, empty) {
			continue
		}

		// TODO GatewayAPI
	}

	return varr, others, nil
}

// ParseInputs reads multiple documents from `kubectl` output and checks with
// the schema. It also returns the list of unrecognized kinds as the second
// response.
//
// NOTE: This function only decodes a subset of the complete k8s
// ObjectMeta as identified by the fields in model.Meta. This
// would typically only be a problem if a user dumps an configuration
// object with kubectl and then re-ingests it.
func ParseInputs(inputs string) ([]config.Config, []DubboKind, error) {
	return parseInputsImpl(inputs, true)
}
