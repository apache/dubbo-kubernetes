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

type verifyCredentialsCallback func(ctx context.Context, image string, credentials hub.Credentials) error

type credentialsCallback func(registry string) (hub.Credentials, error)

type chooseCredentialHelperCallback func(available []string) (string, error)

type credentialsProvider struct {
	promptForCredentials     credentialsCallback
	verifyCredentials        verifyCredentialsCallback
	promptForCredentialStore chooseCredentialHelperCallback
	credentialLoaders        []credentialsCallback
	authFilePath             string
	transport                http.RoundTripper
}

func (k keyChain) Resolve(resource authn.Resource) (authn.Authenticator, error) {
	return &authn.Basic{
		Username: k.user,
		Password: k.pwd,
	}, nil
}

func checkAuth(ctx context.Context, image string, credentials hub.Credentials, trans http.RoundTripper) error {
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

func WithPromptForCredentials(cbk credentialsCallback) Opt {
	return func(opts *credentialsProvider) {
		opts.promptForCredentials = cbk
	}
}

func WithVerifyCredentials(cbk verifyCredentialsCallback) Opt {
	return func(opts *credentialsProvider) {
		opts.verifyCredentials = cbk
	}
}

func WithPromptForCredentialStore(cbk chooseCredentialHelperCallback) Opt {
	return func(opts *credentialsProvider) {
		opts.promptForCredentialStore = cbk
	}
}

func WithTransport(transport http.RoundTripper) Opt {
	return func(opts *credentialsProvider) {
		opts.transport = transport
	}
}
