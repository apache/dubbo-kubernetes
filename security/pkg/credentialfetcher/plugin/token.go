package plugin

import (
	"k8s.io/klog/v2"
	"os"
	"strings"
)

type KubernetesTokenPlugin struct {
	path string
}

func CreateTokenPlugin() *KubernetesTokenPlugin {
	return &KubernetesTokenPlugin{
		path: "",
	}
}

func (t KubernetesTokenPlugin) GetPlatformCredential() (string, error) {
	if t.path == "" {
		return "", nil
	}
	tok, err := os.ReadFile(t.path)
	if err != nil {
		klog.Warningf("failed to fetch token from file: %v", err)
		return "", nil
	}
	return strings.TrimSpace(string(tok)), nil
}

func (t KubernetesTokenPlugin) GetIdentityProvider() string {
	return ""
}

func (t KubernetesTokenPlugin) Stop() {
}
