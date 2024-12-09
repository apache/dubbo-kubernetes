package kube

import (
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type clientConfig struct {
	restConfig rest.Config
}

func (c clientConfig) RawConfig() (api.Config, error) {
	panic("implement me")
}

func (c clientConfig) ClientConfig() (*rest.Config, error) {
	panic("implement me")
}

func (c clientConfig) Namespace() (string, bool, error) {
	panic("implement me")
}

func (c clientConfig) ConfigAccess() clientcmd.ConfigAccess {
	panic("implement me")
}

func NewClientConfigForRestConfig(restConfig *rest.Config) clientcmd.ClientConfig {
	return &clientConfig{
		restConfig: *restConfig,
	}
}
