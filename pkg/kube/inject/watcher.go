package inject

import (
	"context"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"github.com/apache/dubbo-kubernetes/pkg/kube/watcher/configmapwatcher"
	"github.com/fsnotify/fsnotify"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"path/filepath"
	"time"
)

type Watcher interface {
	SetHandler(func(*Config, string) error)
	Run(<-chan struct{})
	Get() (*Config, string, error)
}

type fileWatcher struct {
	watcher    *fsnotify.Watcher
	configFile string
	valuesFile string
	handler    func(*Config, string) error
}

type configMapWatcher struct {
	c         *configmapwatcher.Controller
	client    kube.Client
	namespace string
	name      string
	configKey string
	valuesKey string
	handler   func(*Config, string) error
}

func NewFileWatcher(configFile, valuesFile string) (Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	// watch the parent directory of the target files so we can catch
	// symlink updates of k8s ConfigMaps volumes.
	watchDir, _ := filepath.Split(configFile)
	if err := watcher.Add(watchDir); err != nil {
		return nil, fmt.Errorf("could not watch %v: %v", watchDir, err)
	}
	return &fileWatcher{
		watcher:    watcher,
		configFile: configFile,
		valuesFile: valuesFile,
	}, nil
}

func (w *fileWatcher) Run(stop <-chan struct{}) {
	defer w.watcher.Close()
	var timerC <-chan time.Time
	for {
		select {
		case <-timerC:
			timerC = nil
			sidecarConfig, valuesConfig, err := w.Get()
			if err != nil {
				klog.Errorf("update error: %v", err)
				break
			}
			if w.handler != nil {
				if err := w.handler(sidecarConfig, valuesConfig); err != nil {
					klog.Errorf("update error: %v", err)
				}
			}
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			klog.V(2).Infof("Injector watch update: %+v", event)
			// use a timer to debounce configuration updates
			if (event.Has(fsnotify.Write) || event.Has(fsnotify.Create)) && timerC == nil {
				timerC = time.After(watchDebounceDelay)
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			klog.Errorf("Watcher error: %v", err)
		case <-stop:
			return
		}
	}
}

func (w *fileWatcher) SetHandler(handler func(*Config, string) error) {
	w.handler = handler
}

func (w *fileWatcher) Get() (*Config, string, error) {
	return loadConfig(w.configFile, w.valuesFile)
}

func (w *configMapWatcher) SetHandler(handler func(*Config, string) error) {
	w.handler = handler
}

func (w *configMapWatcher) Run(stop <-chan struct{}) {
	w.c.Run(stop)
}

func (w *configMapWatcher) Get() (*Config, string, error) {
	cms := w.client.Kube().CoreV1().ConfigMaps(w.namespace)
	cm, err := cms.Get(context.TODO(), w.name, metav1.GetOptions{})
	if err != nil {
		return nil, "", err
	}
	return readConfigMap(cm, w.configKey, w.valuesKey)
}

func NewConfigMapWatcher(client kube.Client, namespace, name, configKey, valuesKey string) Watcher {
	w := &configMapWatcher{
		client:    client,
		namespace: namespace,
		name:      name,
		configKey: configKey,
		valuesKey: valuesKey,
	}
	w.c = configmapwatcher.NewController(client, namespace, name, func(cm *v1.ConfigMap) {
		sidecarConfig, valuesConfig, err := readConfigMap(cm, configKey, valuesKey)
		if err != nil {
			klog.Warningf("failed to read injection config from ConfigMap: %v", err)
			return
		}
		if w.handler != nil {
			if err := w.handler(sidecarConfig, valuesConfig); err != nil {
				klog.Errorf("update error: %v", err)
			}
		}
	})
	return w
}

func readConfigMap(cm *v1.ConfigMap, configKey, valuesKey string) (*Config, string, error) {
	if cm == nil {
		return nil, "", fmt.Errorf("no ConfigMap found")
	}

	configYaml, exists := cm.Data[configKey]
	if !exists {
		return nil, "", fmt.Errorf("missing ConfigMap config key %q", configKey)
	}
	c, err := unmarshalConfig([]byte(configYaml))
	if err != nil {
		return nil, "", fmt.Errorf("failed reading config: %v. YAML:\n%s", err, configYaml)
	}

	valuesConfig, exists := cm.Data[valuesKey]
	if !exists {
		return nil, "", fmt.Errorf("missing ConfigMap values key %q", valuesKey)
	}
	return c, valuesConfig, nil
}
