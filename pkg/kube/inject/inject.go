package inject

import (
	"fmt"
	"github.com/Masterminds/sprig/v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
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
	return mergedPod, templatePod, nil
}
