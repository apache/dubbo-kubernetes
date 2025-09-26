package cache

import (
	"github.com/apache/dubbo-kubernetes/pkg/queue"
	"github.com/apache/dubbo-kubernetes/pkg/security"
	"github.com/fsnotify/fsnotify"
	"k8s.io/klog/v2"
	"sync"
)

type FileCert struct {
	ResourceName string
	Filename     string
}

type SecretManagerClient struct {
	certWatcher             *fsnotify.Watcher
	caClient                security.Client
	queue                   queue.Delayed
	configOptions           *security.Options
	existingCertificateFile security.SdsCertificateConfig
	stop                    chan struct{}
	caRootPath              string
	certMutex               sync.RWMutex
	secretHandler           func(resourceName string)
	fileCerts               map[FileCert]struct{}
}

func NewSecretManagerClient(caClient security.Client, options *security.Options) (*SecretManagerClient, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	ret := &SecretManagerClient{
		queue:         queue.NewDelayed(queue.DelayQueueBuffer(0)),
		caClient:      caClient,
		configOptions: options,
		existingCertificateFile: security.SdsCertificateConfig{
			CertificatePath:   options.CertChainFilePath,
			PrivateKeyPath:    options.KeyFilePath,
			CaCertificatePath: options.RootCertFilePath,
		},
		certWatcher: watcher,
		fileCerts:   make(map[FileCert]struct{}),
		stop:        make(chan struct{}),
		caRootPath:  options.CARootPath,
	}

	go ret.queue.Run(ret.stop)
	go ret.handleFileWatch()
	return ret, nil
}

func (sc *SecretManagerClient) Close() {
	_ = sc.certWatcher.Close()
	if sc.caClient != nil {
		sc.caClient.Close()
	}
	close(sc.stop)
}

func (sc *SecretManagerClient) handleFileWatch() {
	for {
		select {
		case event, ok := <-sc.certWatcher.Events:
			// Channel is closed.
			if !ok {
				return
			}
			// We only care about updates that change the file content
			if !(isWrite(event) || isRemove(event) || isCreate(event)) {
				continue
			}
			sc.certMutex.RLock()
			resources := make(map[FileCert]struct{})
			for k, v := range sc.fileCerts {
				resources[k] = v
			}
			sc.certMutex.RUnlock()
			klog.Infof("event for file certificate %s : %s, pushing to proxy", event.Name, event.Op.String())
			// If it is remove event - cleanup from file certs so that if it is added again, we can watch.
			// The cleanup should happen first before triggering callbacks, as the callbacks are async and
			// we may get generate call before cleanup is done and we will end up not watching the file.
			if isRemove(event) {
				sc.certMutex.Lock()
				for fc := range sc.fileCerts {
					if fc.Filename == event.Name {
						klog.V(2).Infof("removing file %s from file certs", event.Name)
						delete(sc.fileCerts, fc)
						break
					}
				}
				sc.certMutex.Unlock()
			}
			// Trigger callbacks for all resources referencing this file. This is practically always
			// a single resource.
			for k := range resources {
				if k.Filename == event.Name {
					sc.OnSecretUpdate(k.ResourceName)
				}
			}
		case err, ok := <-sc.certWatcher.Errors:
			// Channel is closed.
			if !ok {
				return
			}
			// TODO numFileSecretFailures
			klog.Errorf("certificate watch error: %v", err)
		}
	}
}

func (sc *SecretManagerClient) OnSecretUpdate(resourceName string) {
	sc.certMutex.RLock()
	defer sc.certMutex.RUnlock()
	if sc.secretHandler != nil {
		sc.secretHandler(resourceName)
	}
}

func isWrite(event fsnotify.Event) bool {
	return event.Has(fsnotify.Write)
}

func isCreate(event fsnotify.Event) bool {
	return event.Has(fsnotify.Create)
}

func isRemove(event fsnotify.Event) bool {
	return event.Has(fsnotify.Remove)
}
