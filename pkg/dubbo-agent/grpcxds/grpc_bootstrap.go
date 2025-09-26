package grpcxds

import (
	"encoding/json"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/file"
	"github.com/apache/dubbo-kubernetes/pkg/util/protomarshal"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"google.golang.org/protobuf/types/known/durationpb"
	"os"
	"path"
	"time"
)

const (
	ServerListenerNamePrefix   = "xds.dubbo.io/grpc/lds/inbound/"
	ServerListenerNameTemplate = ServerListenerNamePrefix + "%s"
)

type FileWatcherCertProviderConfig struct {
	CertificateFile   string          `json:"certificate_file,omitempty"`
	PrivateKeyFile    string          `json:"private_key_file,omitempty"`
	CACertificateFile string          `json:"ca_certificate_file,omitempty"`
	RefreshDuration   json.RawMessage `json:"refresh_interval,omitempty"`
}

type GenerateBootstrapOptions struct {
	XdsUdsPath       string
	DiscoveryAddress string
	CertDir          string
}

type ChannelCreds struct {
	Type   string `json:"type,omitempty"`
	Config any    `json:"config,omitempty"`
}

type Bootstrap struct {
	XDSServers                 []XdsServer                    `json:"xds_servers,omitempty"`
	Node                       *core.Node                     `json:"node,omitempty"`
	CertProviders              map[string]CertificateProvider `json:"certificate_providers,omitempty"`
	ServerListenerNameTemplate string                         `json:"server_listener_resource_name_template,omitempty"`
}

type XdsServer struct {
	ServerURI      string         `json:"server_uri,omitempty"`
	ChannelCreds   []ChannelCreds `json:"channel_creds,omitempty"`
	ServerFeatures []string       `json:"server_features,omitempty"`
}

type CertificateProvider struct {
	PluginName string `json:"plugin_name,omitempty"`
	Config     any    `json:"config,omitempty"`
}

func GenerateBootstrap(opts GenerateBootstrapOptions) (*Bootstrap, error) {
	// TODO direct to CP should use secure channel (most likely JWT + TLS, but possibly allow mTLS)
	serverURI := opts.DiscoveryAddress
	if opts.XdsUdsPath != "" {
		serverURI = fmt.Sprintf("unix:///%s", opts.XdsUdsPath)
	}

	bootstrap := Bootstrap{
		XDSServers: []XdsServer{{
			ServerURI: serverURI,
			// connect locally via agent
			ChannelCreds:   []ChannelCreds{{Type: "insecure"}},
			ServerFeatures: []string{"xds_v3"},
		}},
		ServerListenerNameTemplate: ServerListenerNameTemplate,
	}

	if opts.CertDir != "" {
		// TODO use a more appropriate interval
		refresh, err := protomarshal.Marshal(durationpb.New(15 * time.Minute))
		if err != nil {
			return nil, err
		}

		bootstrap.CertProviders = map[string]CertificateProvider{
			"default": {
				PluginName: "file_watcher",
				Config: FileWatcherCertProviderConfig{
					PrivateKeyFile:    path.Join(opts.CertDir, "key.pem"),
					CertificateFile:   path.Join(opts.CertDir, "cert-chain.pem"),
					CACertificateFile: path.Join(opts.CertDir, "root-cert.pem"),
					RefreshDuration:   refresh,
				},
			},
		}
	}

	return &bootstrap, nil
}

func GenerateBootstrapFile(opts GenerateBootstrapOptions, path string) (*Bootstrap, error) {
	bootstrap, err := GenerateBootstrap(opts)
	if err != nil {
		return nil, err
	}
	jsonData, err := json.MarshalIndent(bootstrap, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := file.AtomicWrite(path, jsonData, os.FileMode(0o644)); err != nil {
		return nil, fmt.Errorf("failed writing to %s: %v", path, err)
	}
	return bootstrap, nil
}
