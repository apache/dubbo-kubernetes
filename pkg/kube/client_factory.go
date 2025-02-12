package kube

import (
	"github.com/apache/dubbo-kubernetes/pkg/laziness"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	diskcached "k8s.io/client-go/discovery/cached/disk"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type clientFactory struct {
	clientConfig    clientcmd.ClientConfig
	expander        laziness.Laziness[meta.RESTMapper]
	mapper          laziness.Laziness[meta.ResettableRESTMapper]
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
			discoveryCacheDir := computeDiscoverCacheDir(filepath.Join(cacheDir, "discovery"), restConfig.Host)
			return diskcached.NewCachedDiscoveryClientForConfig(restConfig, discoveryCacheDir, httpCacheDir, 6*time.Hour)
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
	cf.expander = laziness.NewWithRetry(func() (meta.RESTMapper, error) {
		discoveryClient, err := cf.discoveryClient.Get()
		if err != nil {
			return nil, err
		}
		mapper, err := cf.mapper.Get()
		if err != nil {
			return nil, err
		}
		return restmapper.NewShortcutExpander(mapper, discoveryClient, func(string) {}), nil
	})
	return cf
}

func (c *clientFactory) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	return c.discoveryClient.Get()
}

func (c *clientFactory) RestClient() (*rest.RESTClient, error) {
	clientConfig, err := c.ToRestConfig()
	if err != nil {
		return nil, err
	}
	return rest.RESTClientFor(clientConfig)
}

func (c *clientFactory) ToRestConfig() (*rest.Config, error) {
	restConfig, err := c.clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	return SetRestDefaults(restConfig), nil
}

func (c *clientFactory) ToRestMapper() (meta.RESTMapper, error) {
	return c.expander.Get()
}

func (c *clientFactory) DynamicClient() (dynamic.Interface, error) {
	restConfig, err := c.ToRestConfig()
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(restConfig)
}

func (c *clientFactory) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return c.clientConfig
}

var overlyCautiousIllegalFileCharacters = regexp.MustCompile(`[^(\w/.)]`)

func computeDiscoverCacheDir(dir, host string) string {
	schemelesshost := strings.Replace(strings.Replace(host, "https://", "", 1), "http://", "", 1)
	safehost := overlyCautiousIllegalFileCharacters.ReplaceAllString(schemelesshost, "_")
	return filepath.Join(dir, safehost)
}
