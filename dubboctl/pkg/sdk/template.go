package sdk

import (
	"context"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/filesystem"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk/dubbo"
)

type template struct {
	name    string
	runtime string
	fs      filesystem.Filesystem
}

type Template interface {
	Name() string
	Fullname() string
	Runtime() string
	Write(ctx context.Context, f *dubbo.DubboConfig) error
}
