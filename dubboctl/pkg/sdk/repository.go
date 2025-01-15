package sdk

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/filesystem"
)

type Repository struct {
	Name     string
	Runtimes []Runtime
	fs       filesystem.Filesystem
}

type Runtime struct {
	Name      string
	Templates []Template
}

func (r *Repository) Template(runtimeName, name string) (t Template, err error) {
	runtime, err := r.Runtime(runtimeName)
	if err != nil {
		return
	}
	for _, t := range runtime.Templates {
		if t.Name() == name {
			return t, nil
		}
	}
	return nil, fmt.Errorf("template not found")
}

func (r *Repository) Runtime(name string) (runtime Runtime, err error) {
	if name == "" {
		return Runtime{}, fmt.Errorf("language runtime required")
	}
	for _, runtime = range r.Runtimes {
		if runtime.Name == name {
			return runtime, err
		}
	}
	return Runtime{}, fmt.Errorf("language runtime not found")
}
