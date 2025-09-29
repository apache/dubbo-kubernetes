package authenticate

import (
	"context"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/security"
	"github.com/apache/dubbo-kubernetes/security/pkg/pki/util"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"net/http"
)

const (
	ClientCertAuthenticatorType = "ClientCertAuthenticator"
)

type ClientCertAuthenticator struct{}

var _ security.Authenticator = &ClientCertAuthenticator{}

func (cca *ClientCertAuthenticator) AuthenticatorType() string {
	return ClientCertAuthenticatorType
}

// Authenticate extracts identities from presented client certificates. This
// method assumes that certificate chain has been properly validated before
// this method is called. In other words, this method does not do certificate
// chain validation itself.
func (cca *ClientCertAuthenticator) Authenticate(authCtx security.AuthContext) (*security.Caller, error) {
	if authCtx.GrpcContext != nil {
		return cca.authenticateGrpc(authCtx.GrpcContext)
	}
	if authCtx.Request != nil {
		return cca.authenticateHTTP(authCtx.Request)
	}
	return nil, nil
}

func (cca *ClientCertAuthenticator) authenticateGrpc(ctx context.Context) (*security.Caller, error) {
	peer, ok := peer.FromContext(ctx)
	if !ok || peer.AuthInfo == nil {
		return nil, fmt.Errorf("no client certificate is presented")
	}

	if authType := peer.AuthInfo.AuthType(); authType != "tls" {
		return nil, fmt.Errorf("unsupported auth type: %q", authType)
	}

	tlsInfo := peer.AuthInfo.(credentials.TLSInfo)
	chains := tlsInfo.State.VerifiedChains
	if len(chains) == 0 || len(chains[0]) == 0 {
		return nil, fmt.Errorf("no verified chain is found")
	}

	ids, err := util.ExtractIDs(chains[0][0].Extensions)
	if err != nil {
		return nil, err
	}

	return &security.Caller{
		AuthSource: security.AuthSourceClientCertificate,
		Identities: ids,
	}, nil
}

func (cca *ClientCertAuthenticator) authenticateHTTP(req *http.Request) (*security.Caller, error) {
	if req.TLS == nil || req.TLS.VerifiedChains == nil {
		return nil, fmt.Errorf("no client certificate is presented")
	}

	chains := req.TLS.VerifiedChains
	if len(chains) == 0 || len(chains[0]) == 0 {
		return nil, fmt.Errorf("no verified chain is found")
	}

	ids, err := util.ExtractIDs(chains[0][0].Extensions)
	if err != nil {
		return nil, err
	}

	return &security.Caller{
		AuthSource: security.AuthSourceClientCertificate,
		Identities: ids,
	}, nil
}
