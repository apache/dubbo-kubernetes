package grpcxds

import (
	"encoding/json"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/model"
	"os"
	"path"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/file"
	"github.com/apache/dubbo-kubernetes/pkg/util/protomarshal"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	ServerListenerNamePrefix    = "xds.dubbo.io/grpc/lds/inbound/"
	ServerListenerNameTemplate  = ServerListenerNamePrefix + "%s"
	FileWatcherCertProviderName = "file_watcher"
)

type FileWatcherCertProviderConfig struct {
	CertificateFile   string          `json:"certificate_file,omitempty"`
	PrivateKeyFile    string          `json:"private_key_file,omitempty"`
	CACertificateFile string          `json:"ca_certificate_file,omitempty"`
	RefreshDuration   json.RawMessage `json:"refresh_interval,omitempty"`
}

type GenerateBootstrapOptions struct {
	Node             *model.Node
	XdsUdsPath       string
	DiscoveryAddress string
	CertDir          string
}

func (cp *CertificateProvider) UnmarshalJSON(data []byte) error {
	var dat map[string]*json.RawMessage
	if err := json.Unmarshal(data, &dat); err != nil {
		return err
	}
	*cp = CertificateProvider{}

	if pluginNameVal, ok := dat["plugin_name"]; ok {
		if err := json.Unmarshal(*pluginNameVal, &cp.PluginName); err != nil {
			return fmt.Errorf("failed parsing plugin_name in certificate_provider: %v", err)
		}
	} else {
		return fmt.Errorf("did not find plugin_name in certificate_provider")
	}

	if configVal, ok := dat["config"]; ok {
		var err error
		switch cp.PluginName {
		case FileWatcherCertProviderName:
			config := FileWatcherCertProviderConfig{}
			err = json.Unmarshal(*configVal, &config)
			cp.Config = config
		default:
			config := FileWatcherCertProviderConfig{}
			err = json.Unmarshal(*configVal, &config)
			cp.Config = config
		}
		if err != nil {
			return fmt.Errorf("failed parsing config in certificate_provider: %v", err)
		}
	} else {
		return fmt.Errorf("did not find config in certificate_provider")
	}

	return nil
}

func (c *FileWatcherCertProviderConfig) FilePaths() []string {
	return []string{c.CertificateFile, c.PrivateKeyFile, c.CACertificateFile}
}

func (b *Bootstrap) FileWatcherProvider() *FileWatcherCertProviderConfig {
	if b == nil || b.CertProviders == nil {
		return nil
	}
	for _, provider := range b.CertProviders {
		if provider.PluginName == FileWatcherCertProviderName {
			cfg, ok := provider.Config.(FileWatcherCertProviderConfig)
			if !ok {
				return nil
			}
			return &cfg
		}
	}
	return nil
}

func LoadBootstrap(file string) (*Bootstrap, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	b := &Bootstrap{}
	if err := json.Unmarshal(data, b); err != nil {
		return nil, err
	}
	return b, err
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
	xdsMeta, err := extractMeta(opts.Node)
	if err != nil {
		return nil, fmt.Errorf("failed extracting xds metadata: %v", err)
	}

	//	// TODO direct to CP should use secure channel (most likely JWT + TLS, but possibly allow mTLS)
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
		Node: &core.Node{
			Id:       opts.Node.ID,
			Locality: opts.Node.Locality,
			Metadata: xdsMeta,
		},
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

func extractMeta(node *model.Node) (*structpb.Struct, error) {
	bytes, err := json.Marshal(node.Metadata)
	if err != nil {
		return nil, err
	}
	rawMeta := map[string]any{}
	if err := json.Unmarshal(bytes, &rawMeta); err != nil {
		return nil, err
	}
	xdsMeta, err := structpb.NewStruct(rawMeta)
	if err != nil {
		return nil, err
	}
	return xdsMeta, nil
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
