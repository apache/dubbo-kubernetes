package kube

import (
	"github.com/apache/dubbo-kubernetes/pkg/laziness"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	diskcached "k8s.io/client-go/discovery/cached/disk"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
	"time"
)

type clientFactory struct {
	clientConfig    clientcmd.ClientConfig
	mapper          laziness.Laziness[meta.RESTMapper]
	discoveryClient laziness.Laziness[discovery.CachedDiscoveryInterface]
}

func newClientFactory(clientConfig clientcmd.ClientConfig, diskCache bool) *clientFactory {
	cf := &clientFactory{
		clientConfig: clientConfig,
	}
	cf.discoveryClient = laziness.NewWithRetry(func() (discovery.CachedDiscoveryInterface, error) {
		restConfig, err := cf.ToRestConfig()
		if err != nil {
			return nil, err
		}
		if diskCache {
			cacheDir := filepath.Join(homedir.HomeDir(), ".kube", "cache")
			httpCacheDir := filepath.Join(cacheDir, "http")
			return diskcached.NewCachedDiscoveryClientForConfig(restConfig, nil, httpCacheDir, 6*time.Hour)
		}
		d, err := discovery.NewDiscoveryClientForConfig(restConfig)
		if err != nil {
			return nil, err
		}
		return memory.NewMemCacheClient(d), nil
	})
	cf.mapper = laziness.NewWithRetry(func() (meta.ResettableRESTMapper, error) {
		discoveryClient, err := cf.ToDiscoveryClient()
		if err != nil {
			return nil, err
		}
		return restmapper.NewDeferredDiscoveryRESTMapper(discoveryClient), nil
	})
	return cf
}

func (c *clientFactory) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	return c.discoveryClient.Get()
}

func (c *clientFactory) ToRestConfig() (*rest.Config, error) {
	restConfig, err := c.clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	return SetRestDefaults(restConfig), nil
}
