package adsc

import (
	"crypto/tls"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/schema/collections"
	"github.com/apache/dubbo-kubernetes/pkg/security"
	"strings"
)

func getClientCertFn(config *Config) func(requestInfo *tls.CertificateRequestInfo) (*tls.Certificate, error) {
	if config.SecretManager != nil {
		return func(requestInfo *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			key, err := config.SecretManager.GenerateSecret(security.WorkloadKeyCertResourceName)
			if err != nil {
				return nil, err
			}
			clientCert, err := tls.X509KeyPair(key.CertificateChain, key.PrivateKey)
			if err != nil {
				return nil, err
			}
			return &clientCert, nil
		}
	}
	if config.CertDir != "" {
		return func(requestInfo *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			certName := config.CertDir + "/cert-chain.pem"
			clientCert, err := tls.LoadX509KeyPair(certName, config.CertDir+"/key.pem")
			if err != nil {
				return nil, err
			}
			return &clientCert, nil
		}
	}

	return nil
}

func convertTypeURLToMCPGVK(typeURL string) (config.GroupVersionKind, bool) {
	parts := strings.SplitN(typeURL, "/", 3)
	if len(parts) != 3 {
		return config.GroupVersionKind{}, false
	}

	gvk := config.GroupVersionKind{
		Group:   parts[0],
		Version: parts[1],
		Kind:    parts[2],
	}

	_, isMCP := collections.Sail.FindByGroupVersionKind(gvk)
	if isMCP {
		return gvk, true
	}

	return config.GroupVersionKind{}, false
}
