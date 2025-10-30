package dubboagent

import (
	"fmt"
	"strings"

	"github.com/apache/dubbo-kubernetes/pkg/security"
	"github.com/apache/dubbo-kubernetes/security/pkg/nodeagent/caclient/providers/aegis"
	"k8s.io/klog/v2"
)

type RootCertProvider interface {
	GetKeyCertsForCA() (string, string)
	FindRootCAForCA() (string, error)
}

var providers = make(map[string]func(*security.Options, RootCertProvider) (security.Client, error))

func init() {
	providers["Aegis"] = createAegis
}

func createAegis(opts *security.Options, a RootCertProvider) (security.Client, error) {
	var tlsOpts *aegis.TLSOptions
	var err error

	if strings.HasSuffix(opts.CAEndpoint, ":15010") {
		klog.Warning("Debug mode or IP-secure network")
	} else {
		tlsOpts = &aegis.TLSOptions{}
		tlsOpts.RootCert, err = a.FindRootCAForCA()
		if err != nil {
			return nil, fmt.Errorf("failed to find root CA cert for CA: %v", err)
		}

		if tlsOpts.RootCert == "" {
			klog.Infof("Using CA %s cert with system certs", opts.CAEndpoint)
		} else if !fileExists(tlsOpts.RootCert) {
			klog.Fatalf("invalid config - %s missing a root certificate %s", opts.CAEndpoint, tlsOpts.RootCert)
		} else {
			klog.Infof("Using CA %s cert with certs: %s", opts.CAEndpoint, tlsOpts.RootCert)
		}

		tlsOpts.Key, tlsOpts.Cert = a.GetKeyCertsForCA()
	}

	tlsOpts.Key, tlsOpts.Cert = a.GetKeyCertsForCA()

	return aegis.NewAegisClient(opts, tlsOpts)
}

func createCAClient(opts *security.Options, a RootCertProvider) (security.Client, error) {
	provider, ok := providers[opts.CAProviderName]
	if !ok {
		return nil, fmt.Errorf("CA provider %q not registered", opts.CAProviderName)
	}
	return provider(opts, a)
}
