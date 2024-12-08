package cli

import (
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/pointer"
)

type instance struct {
	client map[string]kube.CLIClient
	RootFlags
}

type Context interface {
	CLIClient() (kube.CLIClient, error)
	CLIClientWithRevision(rev string) (kube.CLIClient, error)
}

func NewCLIContext(rootFlags *RootFlags) Context {
	if rootFlags == nil {
		rootFlags = &RootFlags{
			kubeconfig:     pointer.Of[string](""),
			Context:        pointer.Of[string](""),
			namespace:      pointer.Of[string](""),
			dubboNamespace: pointer.Of[string](""),
		}
	}
	return &instance{
		RootFlags: *rootFlags,
	}
}

func (i *instance) CLIClient() (kube.CLIClient, error) {
	return nil, nil
}

func (i *instance) CLIClientWithRevision(rev string) (kube.CLIClient, error) {
	if i.client == nil {
		i.client = make(map[string]kube.CLIClient)
	}

}

func newKubeClientWithRevision(kubeconfig, context, revision string) (kube.CLIClient, error) {
	return nil, nil
}
