package sdk

import (
	"context"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk/dubbo"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/util"
	"path"
)

type template struct {
	name       string
	runtime    string
	repository string
	fs         util.Filesystem
}

type Template interface {
	Name() string
	Fullname() string
	Runtime() string
	Write(ctx context.Context, f *dubbo.DubboConfig) error
}

func (t template) Name() string {
	return t.name
}

func (t template) Fullname() string {
	return t.repository + "/" + t.name
}

func (t template) Runtime() string {
	return t.runtime
}

func (t template) Write(ctx context.Context, dc *dubbo.DubboConfig) error {
	mask := func(p string) bool {
		_, f := path.Split(p)
		return f == "template.yaml"
	}

	return util.CopyFromFS(".", dc.Root, util.NewMaskingFS(mask, t.fs))
}
