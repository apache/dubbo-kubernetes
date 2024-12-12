package manifest

import (
	"encoding/json"
	"github.com/apache/dubbo-kubernetes/operator/pkg/component"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

type Manifest struct {
	*unstructured.Unstructured
	Content string
}

type ManifestSet struct {
	Comps     component.Name
	Manifests []Manifest
}

func FromJSON(j []byte) (Manifest, error) {
	us := &unstructured.Unstructured{}
	if err := json.Unmarshal(j, us); err != nil {
		return Manifest{}, err
	}
	yml, err := yaml.Marshal(us)
	if err != nil {
		return Manifest{}, err
	}
	return Manifest{
		Unstructured: us,
		Content:      string(yml),
	}, nil
}

func FromYAML(y []byte) (Manifest, error) {
	us := &unstructured.Unstructured{}
	if err := yaml.Unmarshal(y, us); err != nil {
		return Manifest{}, err
	}
	return Manifest{
		Unstructured: us,
		Content:      string(y),
	}, nil
}

func Parse(output []string) ([]Manifest, error) {
	result := make([]Manifest, 0, len(output))
	for _, m := range output {
		mf, err := FromYAML([]byte(m))
		if err != nil {
			return nil, err
		}
		if mf.GetObjectKind().GroupVersionKind().Kind == "" {
			continue
		}
		result = append(result, mf)
	}
	return result, nil
}

func ParseMultiple(output string) ([]Manifest, error) {
	return Parse()
}
