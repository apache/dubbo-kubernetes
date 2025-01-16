package sdk

import (
	"context"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/fs"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk/dubbo"
	"path"
)

type template struct {
	name    string
	runtime string
	fs      fs.Filesystem
}

type Template interface {
	Name() string
	Runtime() string
	Write(ctx context.Context, f *dubbo.DubboConfig) error
}

func (t template) Write(ctx context.Context, f *dubbo.DubboConfig) error {
	mask := func(p string) bool {
		_, f := path.Split(p)
		return f == "manifest.yaml"
	}

	return fs.CopyFromFS(".", f.Root, fs.NewMaskingFS(mask, t.fs))
}

func (t template) Name() string {
	return t.name
}

func (t template) Runtime() string {
	return t.runtime
}
