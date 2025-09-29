package bootstrap

import (
	"crypto/tls"
	sec_model "github.com/apache/dubbo-kubernetes/pkg/model"
	"k8s.io/klog/v2"
	"log"
	"net/http"
	"strings"
)

type httpServerErrorLogWriter struct{}

func (*httpServerErrorLogWriter) Write(p []byte) (int, error) {
	m := strings.TrimSuffix(string(p), "\n")
	if strings.HasPrefix(m, "http: TLS handshake error") && strings.HasSuffix(m, ": EOF") {
		klog.V(2).Info(m)
	} else {
		klog.Info(m)
	}
	return len(p), nil
}

func (s *Server) initSecureWebhookServer(args *SailArgs) {
	// create the https server for hosting the k8s injectionWebhook handlers.
	if args.ServerOptions.HTTPSAddr == "" {
		s.httpsMux = s.httpMux
		klog.Infof("HTTPS port is disabled, multiplexing webhooks on the httpAddr %v", args.ServerOptions.HTTPAddr)
		return
	}

	tlsConfig := &tls.Config{
		GetCertificate: s.getDubbodCertificate,
		MinVersion:     tls.VersionTLS12,
		CipherSuites:   args.ServerOptions.TLSOptions.CipherSuits,
	}
	// Compliance for control plane validation and injection webhook server.
	sec_model.EnforceGoCompliance(tlsConfig)

	klog.Info("initializing secure webhook server for dubbod webhooks")
	// create the https server for hosting the k8s injectionWebhook handlers.
	s.httpsMux = http.NewServeMux()
	s.httpsServer = &http.Server{
		Addr:      args.ServerOptions.HTTPSAddr,
		ErrorLog:  log.New(&httpServerErrorLogWriter{}, "", 0),
		Handler:   s.httpsMux,
		TLSConfig: tlsConfig,
	}

	// register istiodReadyHandler on the httpsMux so that readiness can also be checked remotely
	s.httpsMux.HandleFunc("/ready", s.dubbodReadyHandler)
}
