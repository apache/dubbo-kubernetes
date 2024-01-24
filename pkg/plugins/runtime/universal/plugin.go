package universal

import (
	core_plugins "github.com/apache/dubbo-kubernetes/pkg/core/plugins"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

type plugin struct{}

func init() {
	core_plugins.Register(core_plugins.Universal, &plugin{})
}

func (p *plugin) Customize(rt core_runtime.Runtime) error {
	return nil
}
