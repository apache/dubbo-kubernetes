package bootstrap

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"net"
	"os"
	"sort"
	"strings"
)

import (
	"github.com/pkg/errors"

	"google.golang.org/protobuf/proto"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	xds_config "github.com/apache/dubbo-kubernetes/pkg/config/xds"
	bootstrap_config "github.com/apache/dubbo-kubernetes/pkg/config/xds/bootstrap"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
	core_manager "github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model/rest"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_xds "github.com/apache/dubbo-kubernetes/pkg/core/xds"
	"github.com/apache/dubbo-kubernetes/pkg/xds/bootstrap/types"
)

type BootstrapGenerator interface {
	Generate(ctx context.Context, request types.BootstrapRequest) (proto.Message, DubboDpBootstrap, error)
}

func NewDefaultBootstrapGenerator(
	resManager core_manager.ResourceManager,
	serverConfig *bootstrap_config.BootstrapServerConfig,
	proxyConfig xds_config.Proxy,
	dpServerCertFile string,
	authEnabledForProxyType map[string]bool,
	enableReloadableTokens bool,
	hdsEnabled bool,
	defaultAdminPort uint32,
) (BootstrapGenerator, error) {
	hostsAndIps, err := hostsAndIPs()
	if err != nil {
		return nil, err
	}
	if serverConfig.Params.XdsHost != "" && !hostsAndIps[serverConfig.Params.XdsHost] {
		return nil, errors.Errorf("hostname: %s set by DUBBO_BOOTSTRAP_SERVER_PARAMS_XDS_HOST is not available in the DP Server certificate. Available hostnames: %q. Change the hostname or generate certificate with proper hostname.", serverConfig.Params.XdsHost, hostsAndIps.slice())
	}
	return &bootstrapGenerator{
		resManager:              resManager,
		config:                  serverConfig,
		proxyConfig:             proxyConfig,
		xdsCertFile:             dpServerCertFile,
		authEnabledForProxyType: authEnabledForProxyType,
		enableReloadableTokens:  enableReloadableTokens,
		hostsAndIps:             hostsAndIps,
		hdsEnabled:              hdsEnabled,
		defaultAdminPort:        defaultAdminPort,
	}, nil
}

func hostsAndIPs() (SANSet, error) {
	hostsAndIps := map[string]bool{}
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if ok {
				hostsAndIps[ipNet.IP.String()] = true
			}
		}
	}
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	hostsAndIps[hostname] = true
	hostsAndIps["localhost"] = true
	return hostsAndIps, nil
}

type bootstrapGenerator struct {
	resManager              core_manager.ResourceManager
	config                  *bootstrap_config.BootstrapServerConfig
	proxyConfig             xds_config.Proxy
	authEnabledForProxyType map[string]bool
	enableReloadableTokens  bool
	xdsCertFile             string
	hostsAndIps             SANSet
	hdsEnabled              bool
	defaultAdminPort        uint32
}

type SANSet map[string]bool

func (s SANSet) slice() []string {
	sans := []string{}
	for san := range s {
		sans = append(sans, san)
	}
	sort.Strings(sans)
	return sans
}

func hostsAndIPsFromCertFile(dpServerCertFile string) (SANSet, error) {
	certBytes, err := os.ReadFile(dpServerCertFile)
	if err != nil {
		return nil, errors.Wrap(err, "could not read certificate")
	}
	pemCert, _ := pem.Decode(certBytes)
	if pemCert == nil {
		return nil, errors.New("could not parse certificate")
	}
	cert, err := x509.ParseCertificate(pemCert.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse certificate")
	}

	hostsAndIps := map[string]bool{}
	for _, dnsName := range cert.DNSNames {
		hostsAndIps[dnsName] = true
	}
	for _, ip := range cert.IPAddresses {
		hostsAndIps[ip.String()] = true
	}
	return hostsAndIps, nil
}

func (b *bootstrapGenerator) Generate(ctx context.Context, request types.BootstrapRequest) (proto.Message, DubboDpBootstrap, error) {
	if request.ProxyType == "" {
		request.ProxyType = string(mesh_proto.DataplaneProxyType)
	}
	dubboDpBootstrap := DubboDpBootstrap{}
	//if err := b.validateRequest(request); err != nil {
	//	return nil, dubboDpBootstrap, err
	//}
	accessLogSocketPath := request.AccessLogSocketPath
	if accessLogSocketPath == "" {
		accessLogSocketPath = core_xds.AccessLogSocketName(os.TempDir(), request.Name, request.Mesh)
	}
	metricsSocketPath := request.MetricsResources.SocketPath

	if metricsSocketPath == "" {
		metricsSocketPath = core_xds.MetricsHijackerSocketName(os.TempDir(), request.Name, request.Mesh)
	}

	proxyId := core_xds.BuildProxyId(request.Mesh, request.Name)
	params := configParameters{
		Id:                 proxyId.String(),
		AdminAddress:       b.config.Params.AdminAddress,
		AdminAccessLogPath: b.adminAccessLogPath(request.OperatingSystem),
		XdsHost:            b.xdsHost(request),
		XdsPort:            b.config.Params.XdsPort,
		XdsConnectTimeout:  b.config.Params.XdsConnectTimeout.Duration,
		DataplaneToken:     request.DataplaneToken,
		DataplaneTokenPath: request.DataplaneTokenPath,
		DataplaneResource:  request.DataplaneResource,
		Version: &mesh_proto.Version{
			DubboDp: &mesh_proto.DubboDpVersion{
				Version:   request.Version.DubboDp.Version,
				GitTag:    request.Version.DubboDp.GitTag,
				GitCommit: request.Version.DubboDp.GitCommit,
				BuildDate: request.Version.DubboDp.BuildDate,
			},
			Envoy: &mesh_proto.EnvoyVersion{
				Version:           request.Version.Envoy.Version,
				Build:             request.Version.Envoy.Build,
				DubboDpCompatible: request.Version.Envoy.DubboDpCompatible,
			},
		},
		DynamicMetadata:     request.DynamicMetadata,
		DNSPort:             request.DNSPort,
		EmptyDNSPort:        request.EmptyDNSPort,
		ProxyType:           request.ProxyType,
		Features:            request.Features,
		Resources:           request.Resources,
		Workdir:             request.Workdir,
		AccessLogSocketPath: accessLogSocketPath,
		MetricsSocketPath:   metricsSocketPath,
		MetricsCertPath:     request.MetricsResources.CertPath,
		MetricsKeyPath:      request.MetricsResources.KeyPath,
	}

	setAdminPort := func(adminPortFromResource uint32) {
		if adminPortFromResource != 0 {
			params.AdminPort = adminPortFromResource
		} else {
			params.AdminPort = b.defaultAdminPort
		}
	}

	switch mesh_proto.ProxyType(params.ProxyType) {
	case mesh_proto.IngressProxyType:
		zoneIngress, err := b.zoneIngressFor(ctx, request, proxyId)
		if err != nil {
			return nil, dubboDpBootstrap, err
		}
		params.Service = "ingress"
		setAdminPort(zoneIngress.Spec.GetNetworking().GetAdmin().GetPort())
	case mesh_proto.EgressProxyType:
		zoneEgress, err := b.zoneEgressFor(ctx, request, proxyId)
		if err != nil {
			return nil, dubboDpBootstrap, err
		}
		params.Service = "egress"
		setAdminPort(zoneEgress.Spec.GetNetworking().GetAdmin().GetPort())

	default:
		return nil, dubboDpBootstrap, errors.Errorf("unknown proxy type %v", params.ProxyType)
	}
	var err error

	config, err := genConfig(params, b.proxyConfig, b.enableReloadableTokens)
	if err != nil {
		return nil, dubboDpBootstrap, errors.Wrap(err, "failed creating bootstrap conf")
	}
	if err = config.Validate(); err != nil {
		return nil, dubboDpBootstrap, errors.Wrap(err, "Envoy bootstrap config is not valid")
	}
	return config, dubboDpBootstrap, nil
}

func (b *bootstrapGenerator) xdsHost(request types.BootstrapRequest) string {
	if b.config.Params.XdsHost != "" { // XdsHost from config takes precedence over Host from request
		return b.config.Params.XdsHost
	} else {
		return request.Host
	}
}

var DpTokenRequired = errors.New("Dataplane Token is required. Generate token using 'dubboctl generate dataplane-token > /path/file' and provide it via --dataplane-token-file=/path/file argument to Dubbo DP")

var NotCA = errors.New("A data plane proxy is trying to verify the control plane using the certificate which is not a certificate authority (basic constraint 'CA' is set to 'false').\n" +
	"Provide CA that was used to sign a certificate used in the control plane by using 'dubbo-dp run --ca-cert-file=file' or via DUBBO_CONTROL_PLANE_CA_CERT_FILE")

func SANMismatchErr(host string, sans []string) error {
	return errors.Errorf("A data plane proxy is trying to connect to the control plane using %q address, but the certificate in the control plane has the following SANs %q. "+
		"Either change the --cp-address in dubbo-dp to one of those or execute the following steps:\n"+
		"1) Generate a new certificate with the address you are trying to use. It is recommended to use trusted Certificate Authority, but you can also generate self-signed certificates using 'dubboctl generate tls-certificate --type=server --cp-hostname=%s'\n"+
		"2) Set DUBBO_GENERAL_TLS_CERT_FILE and DUBBO_GENERAL_TLS_KEY_FILE or the equivalent in Dubbo CP config file to the new certificate.\n"+
		"3) Restart the control plane to read the new certificate and start dubbo-dp.", host, sans, host)
}

func ISSANMismatchErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.HasPrefix(err.Error(), "A data plane proxy is trying to connect to the control plane using")
}

// caCert gets CA cert that was used to signed cert that DP server is protected with.
// Technically result of this function does not have to be a valid CA.
// When user provides custom cert + key and does not provide --ca-cert-file to dubbo-dp run, this can return just a regular cert

func (b *bootstrapGenerator) adminAccessLogPath(operatingSystem string) string {
	if operatingSystem == "" { // backwards compatibility
		return b.config.Params.AdminAccessLogPath
	}
	if b.config.Params.AdminAccessLogPath == os.DevNull && operatingSystem == "windows" {
		// when AdminAccessLogPath was not explicitly set and DPP OS is Windows we need to set window specific DevNull.
		// otherwise when CP is on Linux, we would set /dev/null which is not valid on Windows.
		return "NUL"
	}
	return b.config.Params.AdminAccessLogPath
}

func (b *bootstrapGenerator) validateRequest(request types.BootstrapRequest) error {
	if b.authEnabledForProxyType[request.ProxyType] && request.DataplaneToken == "" && request.DataplaneTokenPath == "" {
		return DpTokenRequired
	}
	if b.config.Params.XdsHost == "" { // XdsHost takes precedence over Host in the request, so validate only when it is not set
		if !b.hostsAndIps[request.Host] {
			return SANMismatchErr(request.Host, b.hostsAndIps.slice())
		}
	}
	return nil
}

func (b *bootstrapGenerator) zoneEgressFor(ctx context.Context, request types.BootstrapRequest, proxyId *core_xds.ProxyId) (*core_mesh.ZoneEgressResource, error) {
	if request.DataplaneResource != "" {
		res, err := rest.YAML.UnmarshalCore([]byte(request.DataplaneResource))
		if err != nil {
			return nil, err
		}
		zoneEgress, ok := res.(*core_mesh.ZoneEgressResource)
		if !ok {
			return nil, errors.Errorf("invalid resource")
		}
		if err := zoneEgress.Validate(); err != nil {
			return nil, err
		}
		return zoneEgress, nil
	} else {
		zoneEgress := core_mesh.NewZoneEgressResource()
		if err := b.resManager.Get(ctx, zoneEgress, core_store.GetBy(proxyId.ToResourceKey())); err != nil {
			return nil, err
		}
		return zoneEgress, nil
	}
}

func (b *bootstrapGenerator) zoneIngressFor(ctx context.Context, request types.BootstrapRequest, proxyId *core_xds.ProxyId) (*core_mesh.ZoneIngressResource, error) {
	if request.DataplaneResource != "" {
		res, err := rest.YAML.UnmarshalCore([]byte(request.DataplaneResource))
		if err != nil {
			return nil, err
		}
		zoneIngress, ok := res.(*core_mesh.ZoneIngressResource)
		if !ok {
			return nil, errors.Errorf("invalid resource")
		}
		if err := zoneIngress.Validate(); err != nil {
			return nil, err
		}
		return zoneIngress, nil
	} else {
		zoneIngress := core_mesh.NewZoneIngressResource()
		if err := b.resManager.Get(ctx, zoneIngress, core_store.GetBy(proxyId.ToResourceKey())); err != nil {
			return nil, err
		}
		return zoneIngress, nil
	}
}
