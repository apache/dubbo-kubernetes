package cli

import "github.com/apache/dubbo-kubernetes/pkg/kube"

type instance struct {
	client map[string]kube.CLIClient
}

type Context interface {
	CLIClient() (kube.CLIClient, error)
}

func (i *instance) CLIClient() (kube.CLIClient, error) {
	return nil, nil
}
