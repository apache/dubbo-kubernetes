package cli

import "github.com/apache/dubbo-kubernetes/pkg/kube"

type instance struct {
	client map[string]kube.CliClient
}

type Context interface {
	CliClient() (kube.CliClient, error)
}

func (i *instance) CliClient() (kube.CliClient, error) {
	return nil, nil
}
