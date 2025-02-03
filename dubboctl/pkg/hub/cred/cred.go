package cred

import (
	"context"
	"errors"
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/hub"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
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

func CheckAuth(ctx context.Context, image string, credentials hub.Credentials, trans http.RoundTripper) error {
	ref, err := name.ParseReference(image)
	if err != nil {
		return fmt.Errorf("cannot parse image reference: %w", err)
	}

	kc := keyChain{
		user: credentials.Username,
		pwd:  credentials.Password,
	}

	err = remote.CheckPushPermission(ref, kc, trans)
	if err != nil {
		var transportErr *transport.Error
		if errors.As(err, &transportErr) && transportErr.StatusCode == 401 {
			return errors.New("bad credentials")
		}
		return err
	}

	return nil
}
