package caclient

import (
	"context"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/security"
	"google.golang.org/grpc/credentials"
)

type DefaultTokenProvider struct {
	opts *security.Options
}

func NewDefaultTokenProvider(opts *security.Options) credentials.PerRPCCredentials {
	return &DefaultTokenProvider{opts}
}

func (t *DefaultTokenProvider) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	if t == nil {
		return nil, nil
	}
	token, err := t.GetToken()
	if err != nil {
		return nil, err
	}
	if token == "" {
		return nil, nil
	}
	return map[string]string{
		"authorization": "Bearer " + token,
	}, nil
}

// Allow the token provider to be used regardless of transport security; callers can determine whether
// this is safe themselves.
func (t *DefaultTokenProvider) RequireTransportSecurity() bool {
	return false
}

func (t *DefaultTokenProvider) GetToken() (string, error) {
	if t.opts.CredFetcher == nil {
		return "", nil
	}
	token, err := t.opts.CredFetcher.GetPlatformCredential()
	if err != nil {
		return "", fmt.Errorf("fetch platform credential: %v", err)
	}

	return token, nil
}
