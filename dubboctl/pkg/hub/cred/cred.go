package cred

import (
	"context"
	"fmt"
	"github.com/google/go-containerregistry/pkg/authn"
	"net/http"
)

type keyChain struct {
	user string
	pwd  string
}

func (k keyChain) Resolve(resource authn.Resource) (authn.Authenticator, error) {
	return &authn.Basic{
		Username: k.user,
		Password: k.pwd,
	}, nil
}

func CheckAuth(ctx context.Context, image string, credentials docker.Credentials, trans http.RoundTripper) error {
	ref, err := name.ParseReference(image)
	if err != nil {
		return fmt.Errorf("cannot parse image reference: %w", err)
	}

	kc := keyChain{
		user: credentials.Username,
		pwd:  credentials.Password,
	}

	return nil
}
