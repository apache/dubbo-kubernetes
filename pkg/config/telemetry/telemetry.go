//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package telemetry

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/config"
	api "github.com/kdubbo/api/telemetry/v1alpha1"
	clientapi "github.com/kdubbo/client-go/pkg/apis/telemetry/v1alpha1"
)

const (
	LocalTraceProvider = "localtrace"
	OTLPPort           = 4317
)

type Resource struct {
	Name              string
	Namespace         string
	CreationTimestamp time.Time
	Spec              *api.Telemetry
}

type Tag struct {
	Name  string
	Value string
}

type EffectiveTracing struct {
	Configured               bool
	Providers                []string
	Tags                     []Tag
	RandomSamplingPercentage *float64
	DisableSpanReporting     *bool
}

func ResourcesFromConfigs(configs []config.Config) []Resource {
	resources := make([]Resource, 0, len(configs))
	for _, cfg := range configs {
		spec, ok := cfg.Spec.(*api.Telemetry)
		if !ok {
			continue
		}
		resources = append(resources, Resource{
			Name:              cfg.Name,
			Namespace:         cfg.Namespace,
			CreationTimestamp: cfg.CreationTimestamp,
			Spec:              spec,
		})
	}
	return resources
}

func ResourcesFromClient(items []*clientapi.Telemetry) []Resource {
	resources := make([]Resource, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resources = append(resources, Resource{
			Name:              item.Name,
			Namespace:         item.Namespace,
			CreationTimestamp: item.CreationTimestamp.Time,
			Spec:              &item.Spec,
		})
	}
	return resources
}

// Resolve applies Telemetry resources from least to most specific:
// meshlevel, namespace, then matching workload selectors. A field explicitly
// present at a lower level replaces that complete field from its parent.
func Resolve(resources []Resource, meshNamespace, workloadNamespace string, workloadLabels map[string]string) EffectiveTracing {
	sorted := append([]Resource(nil), resources...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if !sorted[i].CreationTimestamp.Equal(sorted[j].CreationTimestamp) {
			return sorted[i].CreationTimestamp.Before(sorted[j].CreationTimestamp)
		}
		if sorted[i].Namespace != sorted[j].Namespace {
			return sorted[i].Namespace < sorted[j].Namespace
		}
		return sorted[i].Name < sorted[j].Name
	})

	result := EffectiveTracing{}
	if meshlevel := firstNamespaceLevel(sorted, meshNamespace); meshlevel != nil {
		apply(&result, meshlevel.Spec)
	}
	if workloadNamespace != meshNamespace {
		if namespace := firstNamespaceLevel(sorted, workloadNamespace); namespace != nil {
			apply(&result, namespace.Spec)
		}
	}
	for i := range sorted {
		resource := &sorted[i]
		if resource.Namespace != workloadNamespace || resource.Spec == nil {
			continue
		}
		selector := resource.Spec.GetSelector().GetMatchLabels()
		if len(selector) == 0 || !matches(selector, workloadLabels) {
			continue
		}
		apply(&result, resource.Spec)
	}
	return result
}

func firstNamespaceLevel(resources []Resource, namespace string) *Resource {
	for i := range resources {
		resource := &resources[i]
		if resource.Namespace == namespace && resource.Spec != nil &&
			len(resource.Spec.GetSelector().GetMatchLabels()) == 0 {
			return resource
		}
	}
	return nil
}

func matches(selector, labels map[string]string) bool {
	for key, value := range selector {
		if labels[key] != value {
			return false
		}
	}
	return true
}

func apply(result *EffectiveTracing, spec *api.Telemetry) {
	for _, tracing := range spec.GetTracing() {
		if tracing == nil {
			continue
		}
		result.Configured = true
		if len(tracing.GetProviders()) > 0 {
			result.Providers = make([]string, 0, len(tracing.GetProviders()))
			for _, provider := range tracing.GetProviders() {
				result.Providers = append(result.Providers, provider.GetName())
			}
		}
		if len(tracing.GetTags()) > 0 {
			result.Tags = make([]Tag, 0, len(tracing.GetTags()))
			for _, tag := range tracing.GetTags() {
				result.Tags = append(result.Tags, Tag{Name: tag.GetName(), Value: tag.GetValue()})
			}
		}
		if sampling := tracing.GetRandomSamplingPercentage(); sampling != nil {
			value := sampling.GetValue()
			result.RandomSamplingPercentage = &value
		}
		if disabled := tracing.GetDisableSpanReporting(); disabled != nil {
			value := disabled.GetValue()
			result.DisableSpanReporting = &value
		}
	}
}

func (t EffectiveTracing) Disabled() bool {
	return t.DisableSpanReporting != nil && *t.DisableSpanReporting
}

func (t EffectiveTracing) Provider() string {
	if len(t.Providers) == 0 {
		return ""
	}
	return t.Providers[0]
}

func (t EffectiveTracing) SamplingPercentage() float64 {
	if t.RandomSamplingPercentage == nil {
		return 100
	}
	return *t.RandomSamplingPercentage
}

func (t EffectiveTracing) SamplingPercentageString() string {
	return strconv.FormatFloat(t.SamplingPercentage(), 'f', -1, 64)
}

func (t EffectiveTracing) SamplingRatioString() string {
	return strconv.FormatFloat(t.SamplingPercentage()/100, 'f', -1, 64)
}

func (t EffectiveTracing) ResourceAttributes() string {
	attributes := make([]string, 0, len(t.Tags))
	for _, tag := range t.Tags {
		attributes = append(attributes, tag.Name+"="+tag.Value)
	}
	return strings.Join(attributes, ",")
}

func (t EffectiveTracing) TagsJSON() string {
	if len(t.Tags) == 0 {
		return ""
	}
	tags := make(map[string]string, len(t.Tags))
	for _, tag := range t.Tags {
		tags[tag.Name] = tag.Value
	}
	data, _ := json.Marshal(tags)
	return string(data)
}

func ProviderEndpoint(provider, meshNamespace string) string {
	service := strings.TrimSpace(provider)
	if strings.EqualFold(service, LocalTraceProvider) {
		service = "tracing"
	}
	if service == "" {
		return ""
	}
	if !strings.Contains(service, ".") {
		service = fmt.Sprintf("%s.%s.svc", service, meshNamespace)
	}
	return fmt.Sprintf("http://%s:%d", service, OTLPPort)
}
