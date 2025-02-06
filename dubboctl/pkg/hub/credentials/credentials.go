package credentials

import (
	"context"
	"errors"
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/hub/pusher"
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

type VerifyCredentialsCallback func(ctx context.Context, image string, credentials pusher.Credentials) error

type CredentialsCallback func(registry string) (pusher.Credentials, error)

type ChooseCredentialHelperCallback func(available []string) (string, error)

type credentialsProvider struct {
	promptForCredentials     CredentialsCallback
	verifyCredentials        VerifyCredentialsCallback
	promptForCredentialStore ChooseCredentialHelperCallback
	credentialLoaders        []CredentialsCallback
	authFilePath             string
	transport                http.RoundTripper
}

func (k keyChain) Resolve(resource authn.Resource) (authn.Authenticator, error) {
	return &authn.Basic{
		Username: k.user,
		Password: k.pwd,
	}, nil
}

func checkAuth(ctx context.Context, image string, credentials pusher.Credentials, trans http.RoundTripper) error {
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

type Opt func(opts *credentialsProvider)

func WithPromptForCredentials(cbk CredentialsCallback) Opt {
	return func(opts *credentialsProvider) {
		opts.promptForCredentials = cbk
	}
}

func WithVerifyCredentials(cbk VerifyCredentialsCallback) Opt {
	return func(opts *credentialsProvider) {
		opts.verifyCredentials = cbk
	}
}

func WithPromptForCredentialStore(cbk ChooseCredentialHelperCallback) Opt {
	return func(opts *credentialsProvider) {
		opts.promptForCredentialStore = cbk
	}
}

func WithTransport(transport http.RoundTripper) Opt {
	return func(opts *credentialsProvider) {
		opts.transport = transport
	}
}
