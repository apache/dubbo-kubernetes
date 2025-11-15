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

package inject

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"text/template"

	common_features "github.com/apache/dubbo-kubernetes/pkg/features"
	"istio.io/api/annotation"
	meshconfig "istio.io/api/mesh/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	"github.com/apache/dubbo-kubernetes/pkg/log"
)

const (
	ProxyContainerName = "dubbo-proxy"
)

type InjectionPolicy string

const (
	InjectionPolicyDisabled InjectionPolicy = "disabled"
	InjectionPolicyEnabled  InjectionPolicy = "enabled"
)

type (
	Template     *corev1.Pod
	RawTemplates map[string]string
	Templates    map[string]*template.Template
)

type ContainerReorder int

const (
	MoveFirst ContainerReorder = iota
	MoveLast
	Remove
)

type Config struct {
	Templates           Templates           `json:"-"`
	RawTemplates        RawTemplates        `json:"templates"`
	InjectedAnnotations map[string]string   `json:"injectedAnnotations"`
	DefaultTemplates    []string            `json:"defaultTemplates"`
	Aliases             map[string][]string `json:"aliases"`
	Policy              InjectionPolicy     `json:"policy"`
}

type TemplateData struct {
	TypeMeta                 metav1.TypeMeta
	DeploymentMeta           types.NamespacedName
	ObjectMeta               metav1.ObjectMeta
	Spec                     corev1.PodSpec
	ProxyConfig              *meshconfig.ProxyConfig
	MeshConfig               *meshconfig.MeshConfig
	Values                   map[string]any
	Revision                 string
	NativeSidecars           bool
	ProxyImage               string
	InboundTrafficPolicyMode string
	CompliancePolicy         string
}

type InjectionStatus struct {
	Containers       []string `json:"containers"`
	Volumes          []string `json:"volumes"`
	ImagePullSecrets []string `json:"imagePullSecrets"`
	Revision         string   `json:"revision"`
}

func RunTemplate(params InjectionParameters) (mergedPod *corev1.Pod, templatePod *corev1.Pod, err error) {
	metadata := &params.pod.ObjectMeta
	meshConfig := params.meshConfig

	if err := validateAnnotations(metadata.GetAnnotations()); err != nil {
		log.Errorf("Injection failed due to invalid annotations: %v", err)
		return nil, nil, err
	}

	strippedPod, err := reinsertOverrides(stripPod(params))
	if err != nil {
		return nil, nil, err
	}

	data := TemplateData{
		TypeMeta:                 params.typeMeta,
		DeploymentMeta:           params.deployMeta,
		ObjectMeta:               strippedPod.ObjectMeta,
		Spec:                     strippedPod.Spec,
		ProxyConfig:              params.proxyConfig,
		MeshConfig:               meshConfig,
		Values:                   params.valuesConfig.asMap,
		Revision:                 params.revision,
		ProxyImage:               getProxyImage(params.valuesConfig.asMap, "mfordjody/proxyadapter:0.3.0-debug"),
		InboundTrafficPolicyMode: InboundTrafficPolicyMode(meshConfig),
		CompliancePolicy:         common_features.CompliancePolicy,
	}

	if params.valuesConfig.asMap == nil {
		return nil, nil, fmt.Errorf("failed to parse values.yaml; check Dubbod logs for errors")
	}

	mergedPod = params.pod
	templatePod = &corev1.Pod{}

	for _, templateName := range selectTemplates(params) {
		parsedTemplate, f := params.templates[templateName]
		if !f {
			return nil, nil, fmt.Errorf("requested template %q not found; have %v",
				templateName, strings.Join(knownTemplates(params.templates), ", "))
		}
		bbuf, err := runTemplate(parsedTemplate, data)
		if err != nil {
			return nil, nil, err
		}
		templatePod, err = applyOverlayYAML(templatePod, bbuf.Bytes())
		if err != nil {
			return nil, nil, fmt.Errorf("failed applying injection overlay: %v", err)
		}
		mergedPod, err = applyOverlayYAML(mergedPod, bbuf.Bytes())
		if err != nil {
			return nil, nil, fmt.Errorf("failed parsing generated injected YAML (check Istio sidecar injector configuration): %v", err)
		}
	}

	return mergedPod, templatePod, nil
}

func FindContainer(name string, containers []corev1.Container) *corev1.Container {
	for i := range containers {
		if containers[i].Name == name {
			return &containers[i]
		}
	}
	return nil
}

func UnmarshalConfig(yml []byte) (Config, error) {
	var injectConfig Config
	if err := yaml.Unmarshal(yml, &injectConfig); err != nil {
		return injectConfig, fmt.Errorf("failed to unmarshal injection template: %v", err)
	}

	var err error
	injectConfig.Templates, err = ParseTemplates(injectConfig.RawTemplates)
	if err != nil {
		return injectConfig, err
	}

	return injectConfig, nil
}

func InboundTrafficPolicyMode(meshConfig *meshconfig.MeshConfig) string {
	switch meshConfig.GetInboundTrafficPolicy().GetMode() {
	case meshconfig.MeshConfig_InboundTrafficPolicy_LOCALHOST:
		return "localhost"
	case meshconfig.MeshConfig_InboundTrafficPolicy_PASSTHROUGH:
		return "passthrough"
	}
	return "passthrough"
}

func runTemplate(tmpl *template.Template, data TemplateData) (bytes.Buffer, error) {
	var res bytes.Buffer
	if err := tmpl.Execute(&res, &data); err != nil {
		log.Errorf("Invalid template: %v", err)
		return bytes.Buffer{}, err
	}

	return res, nil
}

func knownTemplates(t Templates) []string {
	keys := make([]string, 0, len(t))
	for k := range t {
		keys = append(keys, k)
	}
	return keys
}

// getProxyImage extracts the proxy image from values map, following Istio's pattern.
// It checks common paths: global.proxy.image, global.proxyImage, and falls back to default.
func getProxyImage(values map[string]any, defaultImage string) string {
	if values == nil {
		return defaultImage
	}

	// Check global.proxy.image (Istio pattern)
	if global, ok := values["global"].(map[string]any); ok {
		if proxy, ok := global["proxy"].(map[string]any); ok {
			if image, ok := proxy["image"].(string); ok && image != "" {
				return image
			}
		}
		// Check global.proxyImage (alternative pattern)
		if image, ok := global["proxyImage"].(string); ok && image != "" {
			return image
		}
	}

	return defaultImage
}

func selectTemplates(params InjectionParameters) []string {
	if a, f := params.pod.Annotations[annotation.InjectTemplates.Name]; f {
		names := []string{}
		for _, tmplName := range strings.Split(a, ",") {
			name := strings.TrimSpace(tmplName)
			names = append(names, name)
		}
		log.Infof("Using templates from annotation: %v", names)
		return resolveAliases(params, names)
	}
	log.Infof("Using default templates: %v", params.defaultTemplate)
	return resolveAliases(params, params.defaultTemplate)
}

func parseDryTemplate(tmplStr string, funcMap map[string]any) (*template.Template, error) {
	temp := template.New("inject")
	t, err := temp.Funcs(sprig.TxtFuncMap()).Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		log.Infof("Failed to parse template: %v %v\n", err, tmplStr)
		return nil, err
	}

	return t, nil
}

func overwriteClusterInfo(pod *corev1.Pod, params InjectionParameters) {
	c := FindProxy(pod)
	if c == nil {
		return
	}
	if len(params.proxyEnvs) > 0 {
		updateClusterEnvs(c, params.proxyEnvs)
	}
}

func updateClusterEnvs(container *corev1.Container, newKVs map[string]string) {
	envVars := make([]corev1.EnvVar, 0)

	for _, env := range container.Env {
		if _, found := newKVs[env.Name]; !found {
			envVars = append(envVars, env)
		}
	}

	keys := make([]string, 0, len(newKVs))
	for key := range newKVs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		val := newKVs[key]
		envVars = append(envVars, corev1.EnvVar{Name: key, Value: val, ValueFrom: nil})
	}
	container.Env = envVars
}

func stripPod(req InjectionParameters) *corev1.Pod {
	pod := req.pod.DeepCopy()
	prevStatus := injectionStatus(pod)
	if prevStatus == nil {
		return req.pod
	}
	// We found a previous status annotation. Possibly we are re-injecting the pod
	// To ensure idempotency, remove our injected containers first
	for _, c := range prevStatus.Containers {
		pod.Spec.Containers = modifyContainers(pod.Spec.Containers, c, Remove)
	}

	delete(pod.Annotations, annotation.SidecarStatus.Name)

	return pod
}

func potentialPodName(metadata metav1.ObjectMeta) string {
	if metadata.Name != "" {
		return metadata.Name
	}
	if metadata.GenerateName != "" {
		return metadata.GenerateName + "***** (actual name not yet known)"
	}
	return ""
}

func modifyContainers(cl []corev1.Container, name string, modifier ContainerReorder) []corev1.Container {
	containers := []corev1.Container{}
	var match *corev1.Container
	for _, c := range cl {
		if c.Name != name {
			containers = append(containers, c)
		} else {
			match = &c
		}
	}
	if match == nil {
		return containers
	}
	switch modifier {
	case MoveFirst:
		return append([]corev1.Container{*match}, containers...)
	case MoveLast:
		return append(containers, *match)
	case Remove:
		return containers
	default:
		return cl
	}
}

func injectRequired(ignored []string, config *Config, podSpec *corev1.PodSpec, metadata metav1.ObjectMeta) bool { // nolint: lll
	if podSpec.HostNetwork {
		return false
	}

	// skip special kubernetes system namespaces
	for _, namespace := range ignored {
		if metadata.Namespace == namespace {
			return false
		}
	}

	annos := metadata.GetAnnotations()

	var useDefault bool
	var inject bool

	objectSelector := annos["proxyless.dubbo.apache.org/inject"]
	if lbl, labelPresent := metadata.GetLabels()["proxyless.dubbo.apache.org/inject"]; labelPresent {
		// The label is the new API; if both are present we prefer the label
		objectSelector = lbl
	}
	switch objectSelector {
	case "true":
		inject = true
	case "false":
		inject = false
	case "":
		useDefault = true
	default:
		log.Warnf("Invalid value for %s: %q. Only 'true' and 'false' are accepted. Falling back to default injection policy.",
			"proxyless.dubbo.apache.org/inject", objectSelector)
		useDefault = true
	}

	var required bool
	switch config.Policy {
	default: // InjectionPolicyOff
		log.Errorf("Illegal value for autoInject:%s, must be one of [%s,%s]. Auto injection disabled!",
			config.Policy, InjectionPolicyDisabled, InjectionPolicyEnabled)
		required = false
	case InjectionPolicyDisabled:
		if useDefault {
			required = false
		} else {
			required = inject
		}
	case InjectionPolicyEnabled:
		if useDefault {
			required = true
		} else {
			required = inject
		}
	}

	return required
}

func injectionStatus(pod *corev1.Pod) *InjectionStatus {
	var statusBytes []byte
	if pod.ObjectMeta.Annotations != nil {
		if value, ok := pod.ObjectMeta.Annotations[annotation.SidecarStatus.Name]; ok {
			statusBytes = []byte(value)
		}
	}
	if statusBytes == nil {
		return nil
	}

	// default case when injected pod has explicit status
	var iStatus InjectionStatus
	if err := json.Unmarshal(statusBytes, &iStatus); err != nil {
		return nil
	}
	return &iStatus
}

func resolveAliases(params InjectionParameters, names []string) []string {
	ret := []string{}
	for _, name := range names {
		if al, f := params.aliases[name]; f {
			ret = append(ret, al...)
		} else {
			ret = append(ret, name)
		}
	}
	return ret
}

func reinsertOverrides(pod *corev1.Pod) (*corev1.Pod, error) {
	type podOverrides struct {
		Containers     []corev1.Container `json:"containers,omitempty"`
		InitContainers []corev1.Container `json:"initContainers,omitempty"`
	}

	existingOverrides := podOverrides{}
	if annotationOverrides, f := pod.Annotations[annotation.ProxyOverrides.Name]; f {
		if err := json.Unmarshal([]byte(annotationOverrides), &existingOverrides); err != nil {
			return nil, err
		}
	}

	pod = pod.DeepCopy()
	for _, c := range existingOverrides.Containers {
		match := FindContainer(c.Name, pod.Spec.Containers)
		if match != nil {
			continue
		}
		pod.Spec.Containers = append(pod.Spec.Containers, c)
	}

	for _, c := range existingOverrides.InitContainers {
		match := FindContainer(c.Name, pod.Spec.InitContainers)
		if match != nil {
			continue
		}
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, c)
	}

	return pod, nil
}
