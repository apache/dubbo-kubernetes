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

package inject

import (
	"encoding/json"
	"errors"
	"fmt"

	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"
	admissionv1 "k8s.io/api/admission/v1"
	kubeApiAdmissionv1beta1 "k8s.io/api/admission/v1beta1"
	"net/http"
	"os"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	opconfig "github.com/apache/dubbo-kubernetes/dubbooperator/pkg/apis"
	"github.com/apache/dubbo-kubernetes/pkg/config/constants"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/multicluster"
	"github.com/apache/dubbo-kubernetes/pkg/util/protomarshal"
	meshv1alpha1 "github.com/kdubbo/api/mesh/v1alpha1"
	"gomodules.xyz/jsonpatch/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/mergepatch"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"sigs.k8s.io/yaml"
)

const (
	watchDebounceDelay = 100 * time.Millisecond
)

var webhookLog = dubbolog.RegisterScope("webhook", "webhook debugging")

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
)

type Webhook struct {
	mu           sync.RWMutex
	watcher      Watcher
	meshConfig   *meshv1alpha1.MeshGlobalConfig
	env          *model.Environment
	Config       *Config
	valuesConfig ValuesConfig
	revision     string
}

type WebhookParameters struct {
	Watcher      Watcher
	Port         int
	Env          *model.Environment
	Mux          *http.ServeMux
	Revision     string
	MultiCluster multicluster.ComponentBuilder
}

type ValuesConfig struct {
	raw      string
	asStruct *opconfig.Values
	asMap    map[string]any
}

type InjectionParameters struct {
	pod                 *corev1.Pod
	deployMeta          types.NamespacedName
	namespace           *corev1.Namespace
	typeMeta            metav1.TypeMeta
	templates           map[string]*template.Template
	defaultTemplate     []string
	aliases             map[string][]string
	meshGlobalConfig    *meshv1alpha1.MeshGlobalConfig
	proxyConfig         *meshv1alpha1.ProxyConfig
	valuesConfig        ValuesConfig
	revision            string
	proxyEnvs           map[string]string
	injectedAnnotations map[string]string
}

func NewWebhook(p WebhookParameters) (*Webhook, error) {
	if p.Mux == nil {
		return nil, errors.New("expected mux to be passed, but was not passed")
	}

	wh := &Webhook{
		watcher:    p.Watcher,
		meshConfig: p.Env.Mesh(),
		env:        p.Env,
		revision:   p.Revision,
	}

	proxylessConfig, valuesConfig, err := p.Watcher.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get initial configuration: %v", err)
	}
	if err := wh.updateConfig(proxylessConfig, valuesConfig); err != nil {
		return nil, fmt.Errorf("failed to process webhook config: %v", err)
	}

	p.Mux.HandleFunc("/inject", wh.serveInject)
	p.Mux.HandleFunc("/inject/", wh.serveInject)

	p.Env.Watcher.AddMeshHandler(func() {
		wh.mu.Lock()
		wh.meshConfig = p.Env.Mesh()
		wh.mu.Unlock()
	})

	return wh, nil
}

func NewValuesConfig(v string) (ValuesConfig, error) {
	c := ValuesConfig{raw: v}
	valuesStruct := &opconfig.Values{}
	if err := protomarshal.ApplyYAML(v, valuesStruct); err != nil {
		return c, fmt.Errorf("could not parse configuration values: %v", err)
	}
	c.asStruct = valuesStruct

	values := map[string]any{}
	if err := yaml.Unmarshal([]byte(v), &values); err != nil {
		return c, fmt.Errorf("could not parse configuration values: %v", err)
	}
	c.asMap = values
	return c, nil
}

func (wh *Webhook) updateConfig(sidecarConfig *Config, valuesConfig string) error {
	wh.mu.Lock()
	defer wh.mu.Unlock()
	wh.Config = sidecarConfig
	vc, err := NewValuesConfig(valuesConfig)
	if err != nil {
		return fmt.Errorf("failed to create new values config: %v", err)
	}
	wh.valuesConfig = vc
	return nil
}

func (wh *Webhook) serveInject(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := kube.HTTPConfigReader(r); err == nil {
			body = data
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	if len(body) == 0 {
		http.Error(w, "no body found", http.StatusBadRequest)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		http.Error(w, "invalid Content-Type, want `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	path := ""
	if r.URL != nil {
		path = r.URL.Path
	}

	var reviewResponse *kube.AdmissionResponse
	var obj runtime.Object
	var ar *kube.AdmissionReview
	if out, _, err := deserializer.Decode(body, nil, obj); err != nil {
		reviewResponse = toAdmissionResponse(err)
	} else {
		ar, err = kube.AdmissionReviewKubeToAdapter(out)
		if err != nil {
			reviewResponse = toAdmissionResponse(err)
		} else {
			reviewResponse = wh.inject(ar, path)
		}
	}

	response := kube.AdmissionReview{}
	response.Response = reviewResponse
	var responseKube runtime.Object
	var apiVersion string
	if ar != nil {
		apiVersion = ar.APIVersion
		response.TypeMeta = ar.TypeMeta
		if response.Response != nil {
			if ar.Request != nil {
				response.Response.UID = ar.Request.UID
			}
		}
	}
	responseKube = kube.AdmissionReviewAdapterToKube(&response, apiVersion)
	resp, err := json.Marshal(responseKube)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(resp); err != nil {
		webhookLog.Errorf("Could not write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
		return
	}
}

func (wh *Webhook) inject(ar *kube.AdmissionReview, path string) *kube.AdmissionResponse {
	log := webhookLog.WithLabels("path", path)
	req := ar.Request
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		log.Errorf("Could not unmarshal raw object: %v %s", err, string(req.Object.Raw))
		return toAdmissionResponse(err)
	}

	pod.ManagedFields = nil

	podName := potentialPodName(pod.ObjectMeta)
	if pod.ObjectMeta.Namespace == "" {
		pod.ObjectMeta.Namespace = req.Namespace
	}

	log = log.WithLabels("pod", pod.Namespace+"/"+podName)
	log.Infof("Process proxyless injection request")

	wh.mu.RLock()
	if !injectRequired(IgnoredNamespaces.UnsortedList(), wh.Config, &pod.Spec, pod.ObjectMeta) {
		log.Infof("Skipping due to policy check")
		wh.mu.RUnlock()
		return &kube.AdmissionResponse{
			Allowed: true,
		}
	}
	proxyConfig := wh.env.GetProxyConfigOrDefault(pod.Namespace, pod.Labels, pod.Annotations, wh.meshConfig)
	deploy, typeMeta := kube.GetDeployMetaFromPod(&pod)
	params := InjectionParameters{
		pod:                 &pod,
		deployMeta:          deploy,
		typeMeta:            typeMeta,
		templates:           wh.Config.Templates,
		defaultTemplate:     wh.Config.DefaultTemplates,
		aliases:             wh.Config.Aliases,
		meshGlobalConfig:    wh.meshConfig,
		proxyConfig:         proxyConfig,
		valuesConfig:        wh.valuesConfig,
		injectedAnnotations: wh.Config.InjectedAnnotations,
		proxyEnvs:           parseInjectEnvs(path),
		revision:            wh.revision,
	}

	wh.mu.RUnlock()

	log.Infof("Injecting pod with templates: %v, defaultTemplate: %v", len(params.templates), params.defaultTemplate)
	log.Infof("Pod labels: %v, annotations: %v", params.pod.Labels, params.pod.Annotations)
	patchBytes, err := injectPod(params)
	if err != nil {
		log.Errorf("Pod injection failed: %v", err)
		return toAdmissionResponse(err)
	}

	log.Infof("Injection Successfully, patch size: %d bytes", len(patchBytes))
	reviewResponse := kube.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *string {
			pt := "JSONPatch"
			return &pt
		}(),
	}
	return &reviewResponse
}

func (wh *Webhook) Run(stop <-chan struct{}) {
	go wh.watcher.Run(stop)
}

func loadConfig(injectFile, valuesFile string) (*Config, string, error) {
	data, err := os.ReadFile(injectFile)
	if err != nil {
		return nil, "", err
	}
	var c *Config
	if c, err = unmarshalConfig(data); err != nil {
		webhookLog.Warnf("Failed to parse injectFile %s", string(data))
		return nil, "", err
	}

	valuesConfig, err := os.ReadFile(valuesFile)
	if err != nil {
		return nil, "", err
	}
	return c, string(valuesConfig), nil
}

func unmarshalConfig(data []byte) (*Config, error) {
	c, err := UnmarshalConfig(data)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func ParseTemplates(tmpls RawTemplates) (Templates, error) {
	ret := make(Templates, len(tmpls))
	for k, t := range tmpls {
		p, err := parseDryTemplate(t, InjectionFuncmap)
		if err != nil {
			return nil, err
		}
		ret[k] = p
	}
	return ret, nil
}

func toAdmissionResponse(err error) *kube.AdmissionResponse {
	return &kube.AdmissionResponse{Result: &metav1.Status{Message: err.Error()}}
}

func injectPod(req InjectionParameters) ([]byte, error) {
	originalPodSpec, err := json.Marshal(req.pod)
	if err != nil {
		return nil, err
	}

	mergedPod, injectedPodData, err := RunTemplate(req)
	if err != nil {
		return nil, fmt.Errorf("failed to run injection template: %v", err)
	}

	if err := postProcessPod(mergedPod, *injectedPodData, req); err != nil {
		return nil, fmt.Errorf("failed to process pod: %v", err)
	}

	patch, err := createPatch(mergedPod, originalPodSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to create patch: %v", err)
	}

	return patch, nil
}

func postProcessPod(pod *corev1.Pod, injectedPod corev1.Pod, req InjectionParameters) error {
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}
	if pod.Labels == nil {
		pod.Labels = map[string]string{}
	}

	removeTemplateOnlyContainers(pod, injectedPod, req.pod)
	overwriteClusterInfo(pod, req)

	if shouldInjectProxylessGRPC(req) {
		// Add proxyless gRPC env and shared bootstrap/cert volume to application containers.
		addApplicationContainerConfig(pod, req)
	}

	if err := reorderPod(pod, req); err != nil {
		return err
	}

	return nil
}

func removeTemplateOnlyContainers(pod *corev1.Pod, injectedPod corev1.Pod, originalPod *corev1.Pod) {
	if originalPod == nil {
		return
	}

	originalContainers := map[string]struct{}{}
	for _, container := range originalPod.Spec.Containers {
		originalContainers[container.Name] = struct{}{}
	}

	templateOnlyContainers := map[string]struct{}{}
	for _, container := range injectedPod.Spec.Containers {
		if container.Image != "" {
			continue
		}
		if _, found := originalContainers[container.Name]; !found {
			templateOnlyContainers[container.Name] = struct{}{}
		}
	}
	if len(templateOnlyContainers) == 0 {
		return
	}

	containers := pod.Spec.Containers[:0]
	for _, container := range pod.Spec.Containers {
		if _, remove := templateOnlyContainers[container.Name]; remove {
			continue
		}
		containers = append(containers, container)
	}
	pod.Spec.Containers = containers
}

func addApplicationContainerConfig(pod *corev1.Pod, req InjectionParameters) {
	discoveryAddress := ""
	trustDomain := constants.DefaultClusterLocalDomain
	if req.meshGlobalConfig != nil {
		if cfg := req.meshGlobalConfig.GetDefaultConfig(); cfg != nil {
			discoveryAddress = cfg.GetDiscoveryAddress()
		}
		if req.meshGlobalConfig.GetTrustDomain() != "" {
			trustDomain = req.meshGlobalConfig.GetTrustDomain()
		}
	}
	if discoveryAddress == "" && req.proxyConfig != nil {
		discoveryAddress = req.proxyConfig.GetDiscoveryAddress()
	}

	meta := pod.ObjectMeta
	if req.pod != nil {
		meta = req.pod.ObjectMeta
	}
	secretName := ProxylessGRPCSecretNameForMeta(meta)
	desiredVolume := corev1.Volume{
		Name: ProxylessXDSVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  secretName,
				DefaultMode: func() *int32 { v := int32(420); return &v }(),
			},
		},
	}
	updated := false
	for i := range pod.Spec.Volumes {
		if pod.Spec.Volumes[i].Name == ProxylessXDSVolumeName {
			pod.Spec.Volumes[i] = desiredVolume
			updated = true
			break
		}
	}
	if !updated {
		pod.Spec.Volumes = append(pod.Spec.Volumes, desiredVolume)
	}

	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		if container.Name == "dubbo-proxy" || container.Name == "dubbo-validation" {
			continue
		}

		container.Env = ensureEnvVar(container.Env, corev1.EnvVar{
			Name:  "GRPC_XDS_BOOTSTRAP",
			Value: ProxylessGRPCBootstrapPath,
		})
		container.Env = ensureEnvVar(container.Env, corev1.EnvVar{
			Name:  "GRPC_XDS_EXPERIMENTAL_SECURITY_SUPPORT",
			Value: "true",
		})
		container.Env = ensureEnvVar(container.Env, corev1.EnvVar{
			Name:  "DUBBO_GRPC_XDS_CREDENTIALS",
			Value: "true",
		})
		container.Env = ensureEnvVar(container.Env, corev1.EnvVar{
			Name:  "DUBBO_GRPC_XDS_RESOLVER",
			Value: "xds:///",
		})
		container.Env = ensureEnvVar(container.Env, corev1.EnvVar{
			Name:  "DUBBO_META_GENERATOR",
			Value: "grpc",
		})
		container.Env = ensureEnvVar(container.Env, corev1.EnvVar{
			Name:  "DUBBO_META_CLUSTER_ID",
			Value: constants.DefaultClusterName,
		})
		container.Env = ensureEnvVar(container.Env, corev1.EnvVar{
			Name: "DUBBO_META_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"},
			},
		})
		container.Env = ensureEnvVar(container.Env, corev1.EnvVar{
			Name:  "CA_ADDR",
			Value: discoveryAddress,
		})
		container.Env = ensureEnvVar(container.Env, corev1.EnvVar{
			Name:  "DUBBO_META_MESH_ID",
			Value: trustDomain,
		})
		container.Env = ensureEnvVar(container.Env, corev1.EnvVar{
			Name:  "TRUST_DOMAIN",
			Value: trustDomain,
		})
		container.Env = ensureEnvVar(container.Env, corev1.EnvVar{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"},
			},
		})
		container.Env = ensureEnvVar(container.Env, corev1.EnvVar{
			Name: "POD_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"},
			},
		})
		container.Env = ensureEnvVar(container.Env, corev1.EnvVar{
			Name: "INSTANCE_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.podIP"},
			},
		})
		container.Env = ensureEnvVar(container.Env, corev1.EnvVar{
			Name: "SERVICE_ACCOUNT",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.serviceAccountName"},
			},
		})
		container.Env = ensureEnvVar(container.Env, corev1.EnvVar{
			Name: "HOST_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.hostIP"},
			},
		})

		hasProxyVolumeMount := false
		for _, vm := range container.VolumeMounts {
			if vm.Name == ProxylessXDSVolumeName {
				hasProxyVolumeMount = true
				break
			}
		}
		if !hasProxyVolumeMount {
			container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
				Name:      ProxylessXDSVolumeName,
				MountPath: ProxylessXDSMountPath,
				ReadOnly:  true,
			})
			webhookLog.Infof("Injection: Added %s volume mount to application container %s", ProxylessXDSMountPath, container.Name)
		}
	}
}

func ensureEnvVar(envs []corev1.EnvVar, desired corev1.EnvVar) []corev1.EnvVar {
	for _, env := range envs {
		if env.Name == desired.Name {
			return envs
		}
	}
	return append(envs, desired)
}

func shouldInjectProxylessGRPC(req InjectionParameters) bool {
	for _, templateName := range selectTemplates(req) {
		if templateName == ProxylessGRPCTemplateName {
			return true
		}
	}
	return false
}

func reorderPod(pod *corev1.Pod, req InjectionParameters) error {
	// Proxy container should be last to ensure `kubectl exec` and similar commands
	// continue to default to the user's container
	pod.Spec.Containers = modifyContainers(pod.Spec.Containers, ProxyContainerName, MoveLast)
	return nil
}

func createPatch(pod *corev1.Pod, original []byte) ([]byte, error) {
	reinjected, err := json.Marshal(pod)
	if err != nil {
		return nil, err
	}
	p, err := jsonpatch.CreatePatch(original, reinjected)
	if err != nil {
		return nil, err
	}
	return json.Marshal(p)
}

func applyOverlayYAML(target *corev1.Pod, overlayYAML []byte) (*corev1.Pod, error) {
	currentJSON, err := json.Marshal(target)
	if err != nil {
		return nil, err
	}

	pod := corev1.Pod{}
	// Overlay the injected template onto the original podSpec
	patched, err := StrategicMergePatchYAML(currentJSON, overlayYAML, pod)
	if err != nil {
		return nil, fmt.Errorf("strategic merge: %v", err)
	}

	if err := json.Unmarshal(patched, &pod); err != nil {
		return nil, fmt.Errorf("unmarshal patched pod: %v", err)
	}
	return &pod, nil
}

func StrategicMergePatchYAML(originalJSON []byte, patchYAML []byte, dataStruct any) ([]byte, error) {
	schema, err := strategicpatch.NewPatchMetaFromStruct(dataStruct)
	if err != nil {
		return nil, err
	}

	originalMap, err := patchHandleUnmarshal(originalJSON, json.Unmarshal)
	if err != nil {
		return nil, err
	}
	patchMap, err := patchHandleUnmarshal(patchYAML, func(data []byte, v any) error {
		return yaml.Unmarshal(data, v)
	})
	if err != nil {
		return nil, err
	}

	result, err := strategicpatch.StrategicMergeMapPatchUsingLookupPatchMeta(originalMap, patchMap, schema)
	if err != nil {
		return nil, err
	}

	return json.Marshal(result)
}

func patchHandleUnmarshal(j []byte, unmarshal func(data []byte, v any) error) (map[string]any, error) {
	if j == nil {
		j = []byte("{}")
	}

	m := map[string]any{}
	err := unmarshal(j, &m)
	if err != nil {
		return nil, mergepatch.ErrBadJSONDoc
	}
	return m, nil
}

func parseInjectEnvs(path string) map[string]string {
	path = strings.TrimSuffix(path, "/")
	res := func(path string) []string {
		parts := strings.SplitN(path, "/", 3)
		var newRes []string
		if len(parts) == 3 { // If length is less than 3, then the path is simply "/inject".
			if strings.HasPrefix(parts[2], ":ENV:") {
				// Deprecated, not recommended.
				//    Note that this syntax fails validation when used to set injectionPath (i.e., service.path in mwh).
				//    It doesn't fail validation when used to set injectionURL, however. K8s bug maybe?
				pairs := strings.Split(parts[2], ":ENV:")
				for i := 1; i < len(pairs); i++ { // skip the first part, it is a nil
					pair := strings.SplitN(pairs[i], "=", 2)
					// The first part is the variable name which can not be empty
					// the second part is the variable value which can be empty but has to exist
					// for example, aaa=bbb, aaa= are valid, but =aaa or = are not valid, the
					// invalid ones will be ignored.
					if len(pair[0]) > 0 && len(pair) == 2 {
						newRes = append(newRes, pair...)
					}
				}
				return newRes
			}
			newRes = strings.Split(parts[2], "/")
		}
		for i, value := range newRes {
			if i%2 != 0 {
				// Replace --slash-- with / in values.
				newRes[i] = strings.ReplaceAll(value, "--slash--", "/")
			}
		}
		return newRes
	}(path)
	newEnvs := make(map[string]string)

	for i := 0; i < len(res); i += 2 {
		k := res[i]
		if i == len(res)-1 { // ignore the last key without value
			webhookLog.Warnf("Add number of inject env entries, ignore the last key %s\n", k)
			break
		}
	}

	return newEnvs
}

func init() {
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admissionv1.AddToScheme(runtimeScheme)
	_ = kubeApiAdmissionv1beta1.AddToScheme(runtimeScheme)
}
