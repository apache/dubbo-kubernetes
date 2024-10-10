package secrets

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	core_ca "github.com/apache/dubbo-kubernetes/pkg/core/ca"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
)

type CaProvider interface {
	// Get returns all PEM encoded CAs, a list of CAs that were used to generate a secret and an error.
	Get(context.Context, *core_mesh.MeshResource) (*core_xds.CaSecret, []string, error)
}

func NewCaProvider(caManagers core_ca.Managers) (CaProvider, error) {
	return &meshCaProvider{
		caManagers: caManagers,
	}, nil
}

type meshCaProvider struct {
	caManagers     core_ca.Managers
	latencyMetrics *prometheus.SummaryVec
}

// Get retrieves the root CA for a given backend with a default timeout of 10
// seconds.
func (s *meshCaProvider) Get(ctx context.Context, mesh *core_mesh.MeshResource) (*core_xds.CaSecret, []string, error) {
	backend := mesh.GetEnabledCertificateAuthorityBackend()
	if backend == nil {
		return nil, nil, errors.New("CA backend is nil")
	}

	//TODO:对用户自定义超时时间支持
	timeout := 10 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	caManager, exist := s.caManagers[backend.Type]
	if !exist {
		return nil, nil, errors.Errorf("CA manager of type %s not exist", backend.Type)
	}

	var certs [][]byte
	var err error
	func() {
		start := time.Now()
		defer func() {
			s.latencyMetrics.WithLabelValues(backend.GetName()).Observe(float64(time.Since(start).Milliseconds()))
		}()
		certs, err = caManager.GetRootCert(ctx, mesh.GetMeta().GetName(), backend)
	}()
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get root certs")
	}

	return &core_xds.CaSecret{
		PemCerts: certs,
	}, []string{backend.Name}, nil
}
