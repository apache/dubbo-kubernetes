package tpath

import (
	"github.com/apache/dubbo-kubernetes/operator/pkg/util"
	yaml2 "github.com/ghodss/yaml"
	"gopkg.in/yaml.v3"
)

func AddSpecRoot(tree string) (string, error) {
	t, nt := make(map[string]interface{}), make(map[string]interface{})
	if err := yaml.Unmarshal([]byte(tree), &t); err != nil {
		return "", err
	}
	nt["spec"] = t
	out, err := yaml.Marshal(nt)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func GetConfigSubtree(manifest, path string) (string, error) {
	root := make(map[string]interface{})
	if err := yaml2.Unmarshal([]byte(manifest), &root); err != nil {
		return "", err
	}

	nc, _, err := GetPathContext(root, util.PathFromString(path), false)
	if err != nil {
		return "", err
	}
	out, err := yaml2.Marshal(nc.Node)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
