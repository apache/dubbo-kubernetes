package inject

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Masterminds/sprig/v3"
	opconfig "github.com/apache/dubbo-kubernetes/operator/pkg/apis"
	common_features "github.com/apache/dubbo-kubernetes/pkg/features"
	"istio.io/api/annotation"
	meshconfig "istio.io/api/mesh/v1alpha1"
	proxyConfig "istio.io/api/networking/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
	"strings"
	"text/template"
)

type (
	Template     *corev1.Pod
	RawTemplates map[string]string
	Templates    map[string]*template.Template
)

type Config struct {
	Templates           Templates           `json:"-"`
	RawTemplates        RawTemplates        `json:"templates"`
	InjectedAnnotations map[string]string   `json:"injectedAnnotations"`
	DefaultTemplates    []string            `json:"defaultTemplates"`
	Aliases             map[string][]string `json:"aliases"`
}

type ProxylessTemplateData struct {
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

type ProxylessInjectionStatus struct {
	InitContainers   []string `json:"initContainers"`
	Containers       []string `json:"containers"`
	Volumes          []string `json:"volumes"`
	ImagePullSecrets []string `json:"imagePullSecrets"`
	Revision         string   `json:"revision"`
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

func parseDryTemplate(tmplStr string, funcMap map[string]any) (*template.Template, error) {
	temp := template.New("inject")
	t, err := temp.Funcs(sprig.TxtFuncMap()).Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		klog.Infof("Failed to parse template: %v %v\n", err, tmplStr)
		return nil, err
	}

	return t, nil
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

func RunTemplate(params InjectionParameters) (mergedPod *corev1.Pod, templatePod *corev1.Pod, err error) {
	meshConfig := params.meshConfig

	strippedPod, err := reinsertOverrides(stripPod(params))
	if err != nil {
		return nil, nil, err
	}

	data := ProxylessTemplateData{
		TypeMeta:         params.typeMeta,
		DeploymentMeta:   params.deployMeta,
		ObjectMeta:       strippedPod.ObjectMeta,
		Spec:             strippedPod.Spec,
		ProxyConfig:      params.proxyConfig,
		MeshConfig:       meshConfig,
		Values:           params.valuesConfig.asMap,
		Revision:         params.revision,
		ProxyImage:       ProxyImage(params.valuesConfig.asStruct, params.proxyConfig.Image, strippedPod.Annotations),
		CompliancePolicy: common_features.CompliancePolicy,
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

func knownTemplates(t Templates) []string {
	keys := make([]string, 0, len(t))
	for k := range t {
		keys = append(keys, k)
	}
	return keys
}

func runTemplate(tmpl *template.Template, data ProxylessTemplateData) (bytes.Buffer, error) {
	var res bytes.Buffer
	if err := tmpl.Execute(&res, &data); err != nil {
		klog.Errorf("Invalid template: %v", err)
		return bytes.Buffer{}, err
	}

	return res, nil
}

func selectTemplates(params InjectionParameters) []string {
	if a, f := params.pod.Annotations[annotation.InjectTemplates.Name]; f {
		names := []string{}
		for _, tmplName := range strings.Split(a, ",") {
			name := strings.TrimSpace(tmplName)
			names = append(names, name)
		}
		return resolveAliases(params, names)
	}
	return resolveAliases(params, params.defaultTemplate)
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
	for _, c := range prevStatus.InitContainers {
		pod.Spec.InitContainers = modifyContainers(pod.Spec.InitContainers, c, Remove)
	}

	delete(pod.Annotations, annotation.SidecarStatus.Name)

	return pod
}

type ContainerReorder int

const (
	MoveFirst ContainerReorder = iota
	MoveLast
	Remove
)

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

func injectionStatus(pod *corev1.Pod) *ProxylessInjectionStatus {
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
	var iStatus ProxylessInjectionStatus
	if err := json.Unmarshal(statusBytes, &iStatus); err != nil {
		return nil
	}
	return &iStatus
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

func FindContainer(name string, containers []corev1.Container) *corev1.Container {
	for i := range containers {
		if containers[i].Name == name {
			return &containers[i]
		}
	}
	return nil
}

func ProxyImage(values *opconfig.Values, image *proxyConfig.ProxyImage, annotations map[string]string) string {
	imageName := "proxyxds"
	return imageName
}
