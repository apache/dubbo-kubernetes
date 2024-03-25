/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package metadata

import (
	"strconv"
	"strings"
	"time"
)

import (
	"github.com/pkg/errors"
)

const (
	// DubboMeshAnnotation defines a Pod annotation that
	// associates a given Pod with a particular Mesh.
	// Annotation value must be the name of a Mesh resource.
	DubboMeshAnnotation = "dubbo.io/mesh"

	// DubboIngressAnnotation allows to mark pod with Dubbo Ingress
	// which is crucial for Multizone communication
	DubboIngressAnnotation = "dubbo.io/ingress"

	// DUBBOSidecarEnvVarsAnnotation is a ; separated list of env vars that will be applied on Kuma Sidecar
	// Example value: TEST1=1;TEST2=2
	DUBBOSidecarEnvVarsAnnotation = "dubbo.io/sidecar-env-vars"

	// DubboEgressAnnotation allows marking pod with Dubbo Egress
	// which is crucial for Multizone communication
	DubboEgressAnnotation = "dubbo.io/egress"

	DubboXdsEnableAnnotation = "dubbo.io/xds-enable"

	// DubboSidecarDrainTime allows to specify drain time of Dubbo DP sidecar.
	DubboSidecarDrainTime = "dubbo.io/sidecar-drain-time"

	// DubboTagsAnnotation holds a JSON representation of desired tags
	DubboTagsAnnotation = "dubbo.io/tags"

	// DubboIngressPublicAddressAnnotation allows to pick public address for Ingress
	// If not defined, Kuma will try to pick this address from the Ingress Service
	DubboIngressPublicAddressAnnotation = "dubbo.io/ingress-public-address"

	// DubboIngressPublicPortAnnotation allows to pick public port for Ingress
	// If not defined, Kuma will try to pick this address from the Ingress Service
	DubboIngressPublicPortAnnotation = "dubbo.io/ingress-public-port"
)

// Annotations that are being automatically set by the Dubbo SDK.
const (
	DubboEnvoyAdminPort            = "dubbo.io/envoy-admin-port"
	DubboSidecarInjectedAnnotation = "dubbo.io/sidecar-injected"
)

const (
	AnnotationEnabled  = "enabled"
	AnnotationDisabled = "disabled"
	AnnotationTrue     = "true"
	AnnotationFalse    = "false"
)

func BoolToEnabled(b bool) string {
	if b {
		return AnnotationEnabled
	}

	return AnnotationDisabled
}

type Annotations map[string]string

func (a Annotations) GetEnabled(keys ...string) (bool, bool, error) {
	return a.GetEnabledWithDefault(false, keys...)
}

func (a Annotations) GetEnabledWithDefault(def bool, keys ...string) (bool, bool, error) {
	v, exists, err := a.getWithDefault(def, func(key, value string) (interface{}, error) {
		switch value {
		case AnnotationEnabled, AnnotationTrue:
			return true, nil
		case AnnotationDisabled, AnnotationFalse:
			return false, nil
		default:
			return false, errors.Errorf("annotation \"%s\" has wrong value \"%s\"", key, value)
		}
	}, keys...)
	if err != nil {
		return def, exists, err
	}
	return v.(bool), exists, nil
}

func (a Annotations) GetUint32(keys ...string) (uint32, bool, error) {
	return a.GetUint32WithDefault(0, keys...)
}

func (a Annotations) GetUint32WithDefault(def uint32, keys ...string) (uint32, bool, error) {
	v, exists, err := a.getWithDefault(def, func(key string, value string) (interface{}, error) {
		u, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return 0, errors.Errorf("failed to parse annotation %q: %s", key, err.Error())
		}
		return uint32(u), nil
	}, keys...)
	if err != nil {
		return def, exists, err
	}
	return v.(uint32), exists, nil
}

func (a Annotations) GetString(keys ...string) (string, bool) {
	return a.GetStringWithDefault("", keys...)
}

func (a Annotations) GetStringWithDefault(def string, keys ...string) (string, bool) {
	v, exists, _ := a.getWithDefault(def, func(key string, value string) (interface{}, error) {
		return value, nil
	}, keys...)
	return v.(string), exists
}

func (a Annotations) GetDurationWithDefault(def time.Duration, keys ...string) (time.Duration, bool, error) {
	v, exists, err := a.getWithDefault(def, func(key string, value string) (interface{}, error) {
		return time.ParseDuration(value)
	}, keys...)
	if err != nil {
		return def, exists, err
	}
	return v.(time.Duration), exists, err
}

func (a Annotations) GetList(keys ...string) ([]string, bool) {
	return a.GetListWithDefault(nil, keys...)
}

func (a Annotations) GetListWithDefault(def []string, keys ...string) ([]string, bool) {
	defCopy := []string{}
	defCopy = append(defCopy, def...)
	v, exists, _ := a.getWithDefault(defCopy, func(key string, value string) (interface{}, error) {
		r := strings.Split(value, ",")
		var res []string
		for _, v := range r {
			if v != "" {
				res = append(res, v)
			}
		}
		return res, nil
	}, keys...)
	return v.([]string), exists
}

// GetMap returns map from annotation. Example: "kuma.io/sidecar-env-vars: TEST1=1;TEST2=2"
func (a Annotations) GetMap(keys ...string) (map[string]string, bool, error) {
	return a.GetMapWithDefault(map[string]string{}, keys...)
}

func (a Annotations) GetMapWithDefault(def map[string]string, keys ...string) (map[string]string, bool, error) {
	defCopy := make(map[string]string, len(def))
	for k, v := range def {
		defCopy[k] = v
	}
	v, exists, err := a.getWithDefault(defCopy, func(key string, value string) (interface{}, error) {
		result := map[string]string{}

		pairs := strings.Split(value, ";")
		for _, pair := range pairs {
			kvSplit := strings.Split(pair, "=")
			if len(kvSplit) != 2 {
				return nil, errors.Errorf("invalid format. Map in %q has to be provided in the following format: key1=value1;key2=value2", key)
			}
			result[kvSplit[0]] = kvSplit[1]
		}
		return result, nil
	}, keys...)
	if err != nil {
		return def, exists, err
	}
	return v.(map[string]string), exists, nil
}

func (a Annotations) getWithDefault(def interface{}, fn func(string, string) (interface{}, error), keys ...string) (interface{}, bool, error) {
	res := def
	exists := false
	for _, k := range keys {
		v, ok := a[k]
		if ok {
			exists = true
			r, err := fn(k, v)
			if err != nil {
				return nil, exists, err
			}
			res = r
		}
	}
	return res, exists, nil
}
