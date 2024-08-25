package certs

import (
	"github.com/apache/dubbo-kubernetes/pkg/api-server/authn"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/plugins"
)

const PluginName = "adminClientCerts"

var log = core.Log.WithName("plugins").WithName("authn").WithName("api-server").WithName("certs")

type plugin struct{}

func init() {
	plugins.Register(PluginName, &plugin{})
}

var _ plugins.AuthnAPIServerPlugin = plugin{}

func (c plugin) NewAuthenticator(_ plugins.PluginContext) (authn.Authenticator, error) {
	log.Info("WARNING: admin client certificates are deprecated. Please migrate to user token as API Server authentication mechanism.")
	return ClientCertAuthenticator, nil
}
