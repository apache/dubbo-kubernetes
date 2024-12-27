package manifest

import (
	"encoding/json"
	"github.com/apache/dubbo-kubernetes/operator/pkg/component"
	"github.com/apache/dubbo-kubernetes/operator/pkg/parts"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

type Manifest struct {
	*unstructured.Unstructured
	Content string
}

type ManifestSet struct {
	Components component.Name
	Manifests  []Manifest
}

func FromJSON(j []byte) (Manifest, error) {
	us := &unstructured.Unstructured{}
	if err := json.Unmarshal(j, us); err != nil {
		return Manifest{}, err
	}
	y, err := yaml.Marshal(us)
	if err != nil {
		return Manifest{}, err
	}
	return Manifest{Unstructured: us, Content: string(y)}, nil
}

func FromYAML(y []byte) (Manifest, error) {
	us := &unstructured.Unstructured{}
	if err := yaml.Unmarshal(y, us); err != nil {
		return Manifest{
			Unstructured: us,
			Content:      string(y),
		}, err
	}
	return Manifest{Unstructured: us, Content: string(y)}, nil
}

func FromObject(us *unstructured.Unstructured) (Manifest, error) {
	c, err := yaml.Marshal(us)
	if err != nil {
		return Manifest{}, err
	}
	return Manifest{
		Unstructured: us,
		Content:      string(c),
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
	return Parse(parts.SplitString(output))
}

func ObjectHash(o *unstructured.Unstructured) string {
	k := o.GroupVersionKind().Kind
	switch o.GroupVersionKind().Kind {
	case "ClusterRole", "ClusterRoleBinding":
		return k + ":" + o.GetName()
	}
	return k + ":" + o.GetNamespace() + ":" + o.GetName()
}

func (m Manifest) Hash() string {
	return ObjectHash(m.Unstructured)
}
