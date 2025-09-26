package grpc

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	sec_model "github.com/apache/dubbo-kubernetes/pkg/model"
	"github.com/apache/dubbo-kubernetes/security/pkg/pki/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"k8s.io/klog/v2"
	"net"
	"os"
	"strings"
)

type TLSOptions struct {
	RootCert      string
	Key           string
	Cert          string
	ServerAddress string
	SAN           string
}

func getTLSDialOption(opts *TLSOptions) (grpc.DialOption, error) {
	rootCert, err := getRootCertificate(opts.RootCert)
	if err != nil {
		return nil, err
	}

	config := tls.Config{
		GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			var certificate tls.Certificate
			key, cert := opts.Key, opts.Cert
			if key != "" && cert != "" {
				isExpired, err := util.IsCertExpired(opts.Cert)
				if err != nil {
					klog.Warningf("cannot parse the cert chain, using token instead: %v", err)
					return &certificate, nil
				}
				if isExpired {
					klog.Warningf("cert expired, using token instead")
					return &certificate, nil
				}
				// Load the certificate from disk
				certificate, err = tls.LoadX509KeyPair(cert, key)
				if err != nil {
					return nil, err
				}
			}
			return &certificate, nil
		},
		RootCAs:    rootCert,
		MinVersion: tls.VersionTLS12,
	}

	if host, _, err := net.SplitHostPort(opts.ServerAddress); err == nil {
		config.ServerName = host
	}
	// For debugging on localhost (with port forward)
	if strings.Contains(config.ServerName, "localhost") {
		config.ServerName = "istiod.istio-system.svc"
	}
	if opts.SAN != "" {
		config.ServerName = opts.SAN
	}
	// Compliance for all gRPC clients (e.g. Citadel)..
	sec_model.EnforceGoCompliance(&config)
	transportCreds := credentials.NewTLS(&config)
	return grpc.WithTransportCredentials(transportCreds), nil
}

func getRootCertificate(rootCertFile string) (*x509.CertPool, error) {
	var certPool *x509.CertPool
	var rootCert []byte
	var err error

	if rootCertFile != "" {
		rootCert, err = os.ReadFile(rootCertFile)
		if err != nil {
			return nil, err
		}

		certPool = x509.NewCertPool()
		ok := certPool.AppendCertsFromPEM(rootCert)
		if !ok {
			return nil, fmt.Errorf("failed to create TLS dial option with root certificates")
		}
	} else {
		certPool, err = x509.SystemCertPool()
		if err != nil {
			return nil, err
		}
	}
	return certPool, nil
}
