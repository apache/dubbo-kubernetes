package cache

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/backoff"
	"github.com/apache/dubbo-kubernetes/pkg/file"
	"github.com/apache/dubbo-kubernetes/pkg/spiffe"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"

	"github.com/apache/dubbo-kubernetes/pkg/queue"
	"github.com/apache/dubbo-kubernetes/pkg/security"
	nodeagentutil "github.com/apache/dubbo-kubernetes/security/pkg/nodeagent/util"
	pkiutil "github.com/apache/dubbo-kubernetes/security/pkg/pki/util"
	"github.com/fsnotify/fsnotify"
	"k8s.io/klog/v2"
)

type FileCert struct {
	ResourceName string
	Filename     string
}

type secretCache struct {
	mu       sync.RWMutex
	workload *security.SecretItem
	certRoot []byte
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
	configTrustBundleMutex  sync.RWMutex
	configTrustBundle       []byte
	outputMutex             sync.Mutex
	generateMutex           sync.Mutex
	cache                   secretCache
}

func (s *secretCache) GetRoot() (rootCert []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.certRoot
}

func (s *secretCache) SetRoot(rootCert []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.certRoot = rootCert
}

func (s *secretCache) GetWorkload() *security.SecretItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.workload == nil {
		return nil
	}
	return s.workload
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

func (sc *SecretManagerClient) GenerateSecret(resourceName string) (secret *security.SecretItem, err error) {
	klog.V(2).InfoS("generate secret %q", resourceName)
	defer func() {
		if secret == nil || err != nil {
			return
		}
		sc.outputMutex.Lock()
		defer sc.outputMutex.Unlock()
		if resourceName == security.RootCertReqResourceName || resourceName == security.WorkloadKeyCertResourceName {
			if err := nodeagentutil.OutputKeyCertToDir(sc.configOptions.OutputKeyCertToDir, secret.PrivateKey,
				secret.CertificateChain, secret.RootCert); err != nil {
				klog.Errorf("error when output the resource: %v", err)
			} else if sc.configOptions.OutputKeyCertToDir != "" {
				klog.V(2).Infof("Output the resource %v to %v", resourceName, sc.configOptions.OutputKeyCertToDir)
			}
		}
	}()

	if sdsFromFile, ns, err := sc.generateFileSecret(resourceName); sdsFromFile {
		if err != nil {
			return nil, err
		}
		return ns, nil
	}

	ns := sc.getCachedSecret(resourceName)
	if ns != nil {
		return ns, nil
	}

	t0 := time.Now()
	sc.generateMutex.Lock()
	defer sc.generateMutex.Unlock()

	// Now that we got the lock, look at cache again before sending request to avoid overwhelming CA
	ns = sc.getCachedSecret(resourceName)
	if ns != nil {
		return ns, nil
	}

	if ts := time.Since(t0); ts > time.Second {
		klog.Warningf("slow generate secret lock: %v", ts)
	}

	if resourceName == security.RootCertReqResourceName {
		// For ROOTCA, directly get root cert bundle from CA client without generating CSR
		if sc.caClient == nil {
			return nil, fmt.Errorf("attempted to fetch root cert, but ca client is nil")
		}
		trustBundlePEM, err := sc.caClient.GetRootCertBundle()
		if err != nil {
			return nil, fmt.Errorf("failed to get root cert bundle: %v", err)
		}
		var rootCertPEM []byte
		if len(trustBundlePEM) > 0 {
			rootCertPEM = concatCerts(trustBundlePEM)
		}
		ns = &security.SecretItem{
			ResourceName: resourceName,
			RootCert:     rootCertPEM,
		}
		ns.RootCert = sc.mergeTrustAnchorBytes(ns.RootCert)
		return ns, nil
	}

	// send request to CA to get new workload certificate
	ns, err = sc.generateNewSecret(resourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to generate workload certificate: %v", err)
	}

	// Store the new secret in the secretCache and trigger the periodic rotation for workload certificate
	sc.registerSecret(*ns)

	// If periodic cert refresh resulted in discovery of a new root, trigger a ROOTCA request to refresh trust anchor
	oldRoot := sc.cache.GetRoot()
	if !bytes.Equal(oldRoot, ns.RootCert) {
		klog.Info("Root cert has changed, start rotating root cert")
		// We store the oldRoot only for comparison and not for serving
		sc.cache.SetRoot(ns.RootCert)
		sc.OnSecretUpdate(security.RootCertReqResourceName)
	}

	return ns, nil
}

func (sc *SecretManagerClient) getCachedSecret(resourceName string) (secret *security.SecretItem) {
	var rootCertBundle []byte
	var ns *security.SecretItem

	if c := sc.cache.GetWorkload(); c != nil {
		if resourceName == security.RootCertReqResourceName {
			rootCertBundle = sc.mergeTrustAnchorBytes(c.RootCert)
			ns = &security.SecretItem{
				ResourceName: resourceName,
				RootCert:     rootCertBundle,
			}
			klog.Infof("returned workload trust anchor from cache (ttl=%v)", time.Until(c.ExpireTime))
		} else {
			ns = &security.SecretItem{
				ResourceName:     resourceName,
				CertificateChain: c.CertificateChain,
				PrivateKey:       c.PrivateKey,
				ExpireTime:       c.ExpireTime,
				CreatedTime:      c.CreatedTime,
			}
			klog.Infof("returned workload certificate from cache (ttl=%v)", time.Until(c.ExpireTime))
		}

		return ns
	}

	return nil
}

func (sc *SecretManagerClient) generateNewSecret(resourceName string) (*security.SecretItem, error) {
	trustBundlePEM := []string{}
	var rootCertPEM []byte

	if sc.caClient == nil {
		return nil, fmt.Errorf("attempted to fetch secret, but ca client is nil")
	}
	t0 := time.Now()

	csrHostName := &spiffe.Identity{
		TrustDomain:    sc.configOptions.TrustDomain,
		Namespace:      sc.configOptions.WorkloadNamespace,
		ServiceAccount: sc.configOptions.ServiceAccount,
	}

	klog.V(2).InfoS("constructed host name for CSR: %s", csrHostName.String())
	options := pkiutil.CertOptions{
		Host:       csrHostName.String(),
		RSAKeySize: sc.configOptions.WorkloadRSAKeySize,
		PKCS8Key:   sc.configOptions.Pkcs8Keys,
		ECSigAlg:   pkiutil.SupportedECSignatureAlgorithms(sc.configOptions.ECCSigAlg),
		ECCCurve:   pkiutil.SupportedEllipticCurves(sc.configOptions.ECCCurve),
	}

	// Generate the cert/key, send CSR to CA.
	csrPEM, keyPEM, err := pkiutil.GenCSR(options)
	if err != nil {
		klog.Errorf(" %s failed to generate key and certificate for CSR: %v", resourceName, err)
		return nil, err
	}

	certChainPEM, err := sc.caClient.CSRSign(csrPEM, int64(sc.configOptions.SecretTTL.Seconds()))
	if err == nil {
		trustBundlePEM, err = sc.caClient.GetRootCertBundle()
	}
	if err != nil {
		klog.Errorf("%s failed to sign: %v", resourceName, err)
		return nil, err
	}

	certChain := concatCerts(certChainPEM)

	var expireTime time.Time
	if expireTime, err = nodeagentutil.ParseCertAndGetExpiryTimestamp(certChain); err != nil {
		klog.Errorf("failed to extract expire time from server certificate in CSR response %+v: %v",
			certChainPEM, err)
		return nil, fmt.Errorf("failed to extract expire time from server certificate in CSR response: %v", err)
	}

	klog.InfoS(
		"Generated new workload certificate",
		"resourceName", resourceName,
		"latency", time.Since(t0),
		"ttl", time.Until(expireTime),
	)

	if len(trustBundlePEM) > 0 {
		rootCertPEM = concatCerts(trustBundlePEM)
	} else {
		// If CA Client has no explicit mechanism to retrieve CA root, infer it from the root of the certChain
		rootCertPEM = []byte(certChainPEM[len(certChainPEM)-1])
	}

	return &security.SecretItem{
		CertificateChain: certChain,
		PrivateKey:       keyPEM,
		ResourceName:     resourceName,
		CreatedTime:      time.Now(),
		ExpireTime:       expireTime,
		RootCert:         rootCertPEM,
	}, nil
}

func concatCerts(certsPEM []string) []byte {
	if len(certsPEM) == 0 {
		return []byte{}
	}
	var certChain bytes.Buffer
	for i, c := range certsPEM {
		certChain.WriteString(c)
		if i < len(certsPEM)-1 && !strings.HasSuffix(c, "\n") {
			certChain.WriteString("\n")
		}
	}
	return certChain.Bytes()
}

func (s *secretCache) SetWorkload(value *security.SecretItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workload = value
}

func (sc *SecretManagerClient) registerSecret(item security.SecretItem) {
	item.ResourceName = security.WorkloadKeyCertResourceName
	// In case there are two calls to GenerateSecret at once, we don't want both to be concurrently registered
	if sc.cache.GetWorkload() != nil {
		klog.V(2).InfoS(
			"Skip scheduling certificate rotation, already scheduled",
			"resource", item.ResourceName,
		)
		return
	}
	sc.cache.SetWorkload(&item)
	klog.V(2).InfoS(
		"Scheduled certificate for rotation",
		"resource", item.ResourceName,
	)
	sc.queue.PushDelayed(func() error {
		// In case `UpdateConfigTrustBundle` called, it will resign workload cert.
		// Check if this is a stale scheduled rotating task.
		if cached := sc.cache.GetWorkload(); cached != nil {
			if cached.CreatedTime == item.CreatedTime {
				klog.V(2).InfoS(
					"Rotating certificate",
					"resource", item.ResourceName,
				)
				sc.cache.SetWorkload(nil)
				sc.OnSecretUpdate(item.ResourceName)
			}
		}
		return nil
	}, 0)
}

func (sc *SecretManagerClient) mergeTrustAnchorBytes(caCerts []byte) []byte {
	return sc.mergeConfigTrustBundle(pkiutil.PemCertBytestoString(caCerts))
}

func (sc *SecretManagerClient) mergeConfigTrustBundle(rootCerts []string) []byte {
	sc.configTrustBundleMutex.RLock()
	existingCerts := pkiutil.PemCertBytestoString(sc.configTrustBundle)
	sc.configTrustBundleMutex.RUnlock()
	anchors := sets.New[string]()
	for _, cert := range existingCerts {
		anchors.Insert(cert)
	}
	for _, cert := range rootCerts {
		anchors.Insert(cert)
	}
	anchorBytes := []byte{}
	for _, cert := range sets.SortedList(anchors) {
		anchorBytes = pkiutil.AppendCertByte(anchorBytes, []byte(cert))
	}
	return anchorBytes
}

func (sc *SecretManagerClient) generateFileSecret(resourceName string) (bool, *security.SecretItem, error) {
	cf := sc.existingCertificateFile
	outputToCertificatePath, ferr := file.DirEquals(filepath.Dir(cf.CertificatePath), sc.configOptions.OutputKeyCertToDir)
	if ferr != nil {
		return false, nil, ferr
	}
	// When there are existing root certificates, or private key and certificate under
	// a well known path, they are used in the SDS response.
	sdsFromFile := false
	var err error
	var sitem *security.SecretItem

	switch {
	// Default root certificate.
	case resourceName == security.RootCertReqResourceName && sc.rootCertificateExist(cf.CaCertificatePath) && !outputToCertificatePath:
		sdsFromFile = true
		if sitem, err = sc.generateRootCertFromExistingFile(cf.CaCertificatePath, resourceName, true); err == nil {
			sitem.RootCert = sc.mergeTrustAnchorBytes(sitem.RootCert)
			sc.addFileWatcher(cf.CaCertificatePath, resourceName)
		}
	// Default workload certificate.
	case resourceName == security.WorkloadKeyCertResourceName && sc.keyCertificateExist(cf.CertificatePath, cf.PrivateKeyPath) && !outputToCertificatePath:
		sdsFromFile = true
		if sitem, err = sc.generateKeyCertFromExistingFiles(cf.CertificatePath, cf.PrivateKeyPath, resourceName); err == nil {
			// Adding cert is sufficient here as key can't change without changing the cert.
			sc.addFileWatcher(cf.CertificatePath, resourceName)
		}
	case resourceName == security.FileRootSystemCACert:
		sdsFromFile = true
		if sc.caRootPath != "" {
			if sitem, err = sc.generateRootCertFromExistingFile(sc.caRootPath, resourceName, false); err == nil {
				sc.addFileWatcher(sc.caRootPath, resourceName)
			}
		} else {
			sdsFromFile = false
		}
	default:
		cfg, ok := security.SdsCertificateConfigFromResourceName(resourceName)
		sdsFromFile = ok
		switch {
		case ok && cfg.IsRootCertificate():
			if sitem, err = sc.generateRootCertFromExistingFile(cfg.CaCertificatePath, resourceName, false); err == nil {
				sc.addFileWatcher(cfg.CaCertificatePath, resourceName)
			}
		case ok && cfg.IsKeyCertificate():
			if sitem, err = sc.generateKeyCertFromExistingFiles(cfg.CertificatePath, cfg.PrivateKeyPath, resourceName); err == nil {
				// Adding cert is sufficient here as key can't change without changing the cert.
				sc.addFileWatcher(cfg.CertificatePath, resourceName)
			}
		}
	}

	if sdsFromFile {
		if err != nil {
			klog.Errorf("failed to generate secret for proxy from file: %v", err)
			return sdsFromFile, nil, err
		}
		klog.V(2).InfoS(
			"Read certificate from file",
			"resource", resourceName,
		)
		return sdsFromFile, sitem, nil
	}
	return sdsFromFile, nil, nil
}

func (sc *SecretManagerClient) keyCertificateExist(certPath, keyPath string) bool {
	b, err := os.ReadFile(certPath)
	if err != nil || len(b) == 0 {
		return false
	}
	b, err = os.ReadFile(keyPath)
	if err != nil || len(b) == 0 {
		return false
	}

	return true
}

var (
	totalTimeout = time.Second * 10
)

func (sc *SecretManagerClient) rootCertificateExist(filePath string) bool {
	b, err := os.ReadFile(filePath)
	if err != nil || len(b) == 0 {
		return false
	}
	return true
}

func (sc *SecretManagerClient) generateKeyCertFromExistingFiles(certChainPath, keyPath, resourceName string) (*security.SecretItem, error) {
	// There is a remote possibility that key is written and cert is not written yet.
	// To handle that case, check if cert and key are valid if they are valid then only send to proxy.
	o := backoff.DefaultOption()
	o.InitialInterval = sc.configOptions.FileDebounceDuration
	b := backoff.NewExponentialBackOff(o)
	secretValid := func() error {
		_, err := tls.LoadX509KeyPair(certChainPath, keyPath)
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), totalTimeout)
	defer cancel()
	if err := b.RetryWithContext(ctx, secretValid); err != nil {
		return nil, err
	}
	return sc.keyCertSecretItem(certChainPath, keyPath, resourceName)
}

func (sc *SecretManagerClient) keyCertSecretItem(cert, key, resource string) (*security.SecretItem, error) {
	certChain, err := sc.readFileWithTimeout(cert)
	if err != nil {
		return nil, err
	}
	keyPEM, err := sc.readFileWithTimeout(key)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var certExpireTime time.Time
	if certExpireTime, err = nodeagentutil.ParseCertAndGetExpiryTimestamp(certChain); err != nil {
		klog.Errorf("failed to extract expiration time in the certificate loaded from file: %v", err)
		return nil, fmt.Errorf("failed to extract expiration time in the certificate loaded from file: %v", err)
	}

	return &security.SecretItem{
		CertificateChain: certChain,
		PrivateKey:       keyPEM,
		ResourceName:     resource,
		CreatedTime:      now,
		ExpireTime:       certExpireTime,
	}, nil
}

const (
	firstRetryBackOffDuration = 50 * time.Millisecond
)

func (sc *SecretManagerClient) readFileWithTimeout(path string) ([]byte, error) {
	retryBackoff := firstRetryBackOffDuration
	timeout := time.After(totalTimeout)
	for {
		cert, err := os.ReadFile(path)
		if err == nil {
			return cert, nil
		}
		select {
		case <-time.After(retryBackoff):
			retryBackoff *= 2
		case <-timeout:
			return nil, err
		case <-sc.stop:
			return nil, err
		}
	}
}

func (sc *SecretManagerClient) generateRootCertFromExistingFile(rootCertPath, resourceName string, workload bool) (*security.SecretItem, error) {
	var rootCert []byte
	var err error
	o := backoff.DefaultOption()
	o.InitialInterval = sc.configOptions.FileDebounceDuration
	b := backoff.NewExponentialBackOff(o)
	certValid := func() error {
		rootCert, err = os.ReadFile(rootCertPath)
		if err != nil {
			return err
		}
		_, _, err := pkiutil.ParsePemEncodedCertificateChain(rootCert)
		if err != nil {
			return err
		}
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), totalTimeout)
	defer cancel()
	if err := b.RetryWithContext(ctx, certValid); err != nil {
		return nil, err
	}

	// Set the rootCert only if it is workload root cert.
	if workload {
		sc.cache.SetRoot(rootCert)
	}
	return &security.SecretItem{
		ResourceName: resourceName,
		RootCert:     rootCert,
	}, nil
}

func (sc *SecretManagerClient) addFileWatcher(file string, resourceName string) {
	if err := sc.tryAddFileWatcher(file, resourceName); err == nil {
		return
	}
	go func() {
		b := backoff.NewExponentialBackOff(backoff.DefaultOption())
		_ = b.RetryWithContext(context.TODO(), func() error {
			err := sc.tryAddFileWatcher(file, resourceName)
			return err
		})
	}()
}

func (sc *SecretManagerClient) tryAddFileWatcher(file string, resourceName string) error {
	// Check if this file is being already watched, if so ignore it. This check is needed here to
	// avoid processing duplicate events for the same file.
	sc.certMutex.Lock()
	defer sc.certMutex.Unlock()
	file, err := filepath.Abs(file)
	if err != nil {
		klog.Errorf("%v: error finding absolute path of %s, retrying watches: %v", resourceName, file, err)
		return err
	}
	key := FileCert{
		ResourceName: resourceName,
		Filename:     file,
	}
	if _, alreadyWatching := sc.fileCerts[key]; alreadyWatching {
		klog.Infof("already watching file for %s", file)
		// Already watching, no need to do anything
		return nil
	}
	sc.fileCerts[key] = struct{}{}
	// File is not being watched, start watching now and trigger key push.
	klog.Infof("adding watcher for file certificate %s", file)
	if err := sc.certWatcher.Add(file); err != nil {
		klog.Errorf("%v: error adding watcher for file %v, retrying watches: %v", resourceName, file, err)
		return err
	}
	return nil
}

func (sc *SecretManagerClient) Close() {
	_ = sc.certWatcher.Close()
	if sc.caClient != nil {
		sc.caClient.Close()
	}
	close(sc.stop)
}

func (sc *SecretManagerClient) RegisterSecretHandler(h func(resourceName string)) {
	sc.certMutex.Lock()
	defer sc.certMutex.Unlock()
	sc.secretHandler = h
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

func (sc *SecretManagerClient) UpdateConfigTrustBundle(trustBundle []byte) error {
	sc.configTrustBundleMutex.Lock()
	if bytes.Equal(sc.configTrustBundle, trustBundle) {
		klog.Info("skip for same trust bundle")
		sc.configTrustBundleMutex.Unlock()
		return nil
	}
	sc.configTrustBundle = trustBundle
	sc.configTrustBundleMutex.Unlock()
	klog.Info("update new trust bundle")
	sc.OnSecretUpdate(security.RootCertReqResourceName)
	sc.OnSecretUpdate(security.WorkloadKeyCertResourceName)
	return nil
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
