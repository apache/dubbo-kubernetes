package client

import (
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/plugins/authn/api-server/tokens/ws"
	"github.com/apache/dubbo-kubernetes/pkg/tokens"
	util_http "github.com/apache/dubbo-kubernetes/pkg/util/http"
)

type UserTokenClient interface {
	Generate(name string, groups []string, validFor time.Duration) (string, error)
}

var _ UserTokenClient = &httpUserTokenClient{}

func NewHTTPUserTokenClient(client util_http.Client) UserTokenClient {
	return &httpUserTokenClient{
		client: tokens.NewTokenClient(client, "user"),
	}
}

type httpUserTokenClient struct {
	client tokens.TokenClient
}

func (h *httpUserTokenClient) Generate(name string, groups []string, validFor time.Duration) (string, error) {
	tokenReq := &ws.UserTokenRequest{
		Name:     name,
		Groups:   groups,
		ValidFor: validFor.String(),
	}
	return h.client.Generate(tokenReq)
}
