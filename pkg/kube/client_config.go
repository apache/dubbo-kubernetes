package kube

import (
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

type clientConfig struct {
	restConfig rest.Config
}

func NewClientConfigForRestConfig(restConfig *rest.Config) clientcmd.ClientConfig {
	return &clientConfig{
		restConfig: *restConfig,
	}
}

func (c clientConfig) RawConfig() (api.Config, error) {
	cfg := api.Config{
		Kind:           "Config",
		APIVersion:     "v1",
		Preferences:    api.Preferences{},
		Clusters:       map[string]*api.Cluster{},
		AuthInfos:      map[string]*api.AuthInfo{},
		Contexts:       map[string]*api.Context{},
		CurrentContext: "",
	}
	return cfg, nil
}

func (c clientConfig) ClientConfig() (*rest.Config, error) {
	return c.copyRestConfig(), nil
}

func (c *clientConfig) copyRestConfig() *rest.Config {
	out := c.restConfig
	return &out
}

func (c clientConfig) Namespace() (string, bool, error) {
	return "default", false, nil
}

func (c clientConfig) ConfigAccess() clientcmd.ConfigAccess {
	return nil
}
