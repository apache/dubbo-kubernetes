package builder

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk/dubbo"
)

const (
	Pack = "pack"
)

type ErrNoDefaultImage struct {
	Builder string
	Runtime string
}

type ErrRuntimeRequired struct {
	Builder string
}

func (e ErrNoDefaultImage) Error() string {
	return fmt.Sprintf("the '%v' runtime defines no default '%v' builder image", e.Runtime, e.Builder)
}

func (e ErrRuntimeRequired) Error() string {
	return fmt.Sprintf("runtime required to choose a default '%v' builder image", e.Builder)
}

func Image(dc *dubbo.DubboConfig, builder string, defaults map[string]string) (string, error) {
	v, ok := dc.Build.BuilderImages[builder]
	if ok {
		return v, nil
	}
	if dc.Runtime == "" {
		return "", ErrRuntimeRequired{Builder: builder}
	}
	v, ok = defaults[dc.Runtime]
	dc.Build.BuilderImages[builder] = v
	if ok {
		return v, nil
	}
	return "", ErrNoDefaultImage{
		Builder: builder,
		Runtime: dc.Runtime,
	}
}
