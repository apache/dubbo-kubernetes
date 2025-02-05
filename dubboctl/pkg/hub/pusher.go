package hub

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"io"
	"net/http"
)

type Opt func(*Pusher)

type Credentials struct {
	Username string
	Password string
}

type CredentialsProvider func(ctx context.Context, image string) (Credentials, error)

type PusherDockerClientFactory func() (PusherDockerClient, error)

type Pusher struct {
	credentialsProvider CredentialsProvider
	transport           http.RoundTripper
	dockerClientFactory PusherDockerClientFactory
}

type PusherDockerClient interface {
	daemon.Client
	ImagePush(ctx context.Context, ref string, options types.ImagePushOptions) (io.ReadCloser, error)
	Close() error
}
