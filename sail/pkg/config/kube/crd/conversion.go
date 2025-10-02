package crd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/resource"
	"io"
	kubeyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/klog/v2"
	"reflect"
)

type ConversionFunc = func(s resource.Schema, js string) (config.Spec, error)

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

func ParseInputs(inputs string) ([]config.Config, []DubboKind, error) {
	return parseInputsImpl(inputs, true)
}

func FromJSON(s resource.Schema, js string) (config.Spec, error) {
	c, err := s.NewInstance()
	if err != nil {
		return nil, err
	}
	if err = config.ApplyJSON(c, js); err != nil {
		return nil, err
	}
	return c, nil
}

func ConvertObject(schema resource.Schema, object DubboObject, domain string) (*config.Config, error) {
	return ConvertObjectInternal(schema, object, domain, FromJSON)
}

func StatusJSONFromMap(schema resource.Schema, jsonMap *json.RawMessage) (config.Status, error) {
	if jsonMap == nil {
		return nil, nil
	}
	js, err := json.Marshal(jsonMap)
	if err != nil {
		return nil, err
	}
	status, err := schema.Status()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(js, status)
	if err != nil {
		return nil, err
	}
	return status, nil
}

func ConvertObjectInternal(schema resource.Schema, object DubboObject, domain string, convert ConversionFunc) (*config.Config, error) {
	js, err := json.Marshal(object.GetSpec())
	if err != nil {
		return nil, err
	}
	spec, err := convert(schema, string(js))
	if err != nil {
		return nil, err
	}
	status, err := StatusJSONFromMap(schema, object.GetStatus())
	if err != nil {
		klog.Errorf("could not get istio status from map %v, err %v", object.GetStatus(), err)
	}
	meta := object.GetObjectMeta()

	return &config.Config{
		Meta: config.Meta{
			GroupVersionKind:  schema.GroupVersionKind(),
			Name:              meta.Name,
			Namespace:         meta.Namespace,
			Domain:            domain,
			Labels:            meta.Labels,
			Annotations:       meta.Annotations,
			ResourceVersion:   meta.ResourceVersion,
			CreationTimestamp: meta.CreationTimestamp.Time,
		},
		Spec:   spec,
		Status: status,
	}, nil
}
