package mirror

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	fnssh "github.com/apache/dubbo-kubernetes/dubboctl/pkg/mirror/ssh"
	"github.com/docker/cli/cli/config"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

var NoDockerAPIError = errors.New("docker API not available")

func NewClient(defaultHost string) (dockerClient client.CommonAPIClient, dockerHostInRemote string, err error) {
	var u *url.URL

	dockerHost := os.Getenv("DOCKER_HOST")
	dockerHostSSHIdentity := os.Getenv("DOCKER_HOST_SSH_IDENTITY")
	hostKeyCallback := fnssh.NewHostKeyCbk()

	if dockerHost == "" {
		u, err = url.Parse(defaultHost)
		if err != nil {
			return
		}
		_, err = os.Stat(u.Path)
		switch {
		case err == nil:
			dockerHost = defaultHost
		case err != nil && !os.IsNotExist(err):
			return
		}
	}

	if dockerHost == "" {
		return nil, "", NoDockerAPIError
	}

	dockerHostInRemote = dockerHost

	u, err = url.Parse(dockerHost)
	isSSH := err == nil && u.Scheme == "ssh"
	isTCP := err == nil && u.Scheme == "tcp"
	isNPipe := err == nil && u.Scheme == "npipe"
	isUnix := err == nil && u.Scheme == "unix"

	if isTCP || isNPipe {
		// With TCP or npipe, it's difficult to determine how to expose the daemon socket to lifecycle containers,
		// so we are defaulting to standard docker location by returning empty string.
		// This should work well most of the time.
		dockerHostInRemote = ""
	}

	if isUnix && runtime.GOOS == "darwin" {
		// A unix socket on macOS is most likely tunneled from VM,
		// so it cannot be mounted under that path.
		dockerHostInRemote = ""
	}

	if !isSSH {
		opts := []client.Opt{client.FromEnv, client.WithAPIVersionNegotiation()}
		if isTCP {
			if httpClient := newHttpClient(); httpClient != nil {
				opts = append(opts, client.WithHTTPClient(httpClient))
			}
		}
		dockerClient, err = client.NewClientWithOpts(opts...)
		return
	}

	credentialsConfig := fnssh.Config{
		Identity:           dockerHostSSHIdentity,
		PassPhrase:         os.Getenv("DOCKER_HOST_SSH_IDENTITY_PASSPHRASE"),
		PasswordCallback:   fnssh.NewPasswordCbk(),
		PassPhraseCallback: fnssh.NewPassPhraseCbk(),
		HostKeyCallback:    hostKeyCallback,
	}
	contextDialer, dockerHostInRemote, err := fnssh.NewDialContext(u, credentialsConfig)
	if err != nil {
		return
	}

	httpClient := &http.Client{
		// No tls
		// No proxy
		Transport: &http.Transport{
			DialContext: contextDialer.DialContext,
		},
	}

	dockerClient, err = client.NewClientWithOpts(
		client.WithAPIVersionNegotiation(),
		client.WithHTTPClient(httpClient),
		client.WithHost("tcp://placeholder/"))

	if closer, ok := contextDialer.(io.Closer); ok {
		dockerClient = clientWithAdditionalCleanup{
			CommonAPIClient: dockerClient,
			cleanUp: func() {
				closer.Close()
			},
		}
	}

	return dockerClient, dockerHostInRemote, err
}

type clientWithAdditionalCleanup struct {
	client.CommonAPIClient
	cleanUp func()
}

func (w clientWithAdditionalCleanup) Close() error {
	defer w.cleanUp()
	return w.CommonAPIClient.Close()
}

func newHttpClient() *http.Client {
	tlsVerifyStr, tlsVerifyChanged := os.LookupEnv("DOCKER_TLS_VERIFY")

	if !tlsVerifyChanged {
		return nil
	}

	var tlsOpts []func(*tls.Config)

	tlsVerify := true
	if b, err := strconv.ParseBool(tlsVerifyStr); err == nil {
		tlsVerify = b
	}

	if !tlsVerify {
		tlsOpts = append(tlsOpts, func(t *tls.Config) {
			t.InsecureSkipVerify = true
		})
	}

	dockerCertPath := os.Getenv("DOCKER_CERT_PATH")
	if dockerCertPath == "" {
		dockerCertPath = config.Dir()
	}

	// Set root CA.
	caData, err := os.ReadFile(filepath.Join(dockerCertPath, "ca.pem"))
	if err == nil {
		certPool := x509.NewCertPool()
		if certPool.AppendCertsFromPEM(caData) {
			tlsOpts = append(tlsOpts, func(t *tls.Config) {
				t.RootCAs = certPool
			})
		}
	}

	// Set client certificate.
	certData, certErr := os.ReadFile(filepath.Join(dockerCertPath, "cert.pem"))
	keyData, keyErr := os.ReadFile(filepath.Join(dockerCertPath, "key.pem"))
	if certErr == nil && keyErr == nil {
		cliCert, err := tls.X509KeyPair(certData, keyData)
		if err == nil {
			tlsOpts = append(tlsOpts, func(cfg *tls.Config) {
				cfg.Certificates = []tls.Certificate{cliCert}
			})
		}
	}

	dialer := &net.Dialer{
		KeepAlive: 30 * time.Second,
		Timeout:   30 * time.Second,
	}

	tlsConfig := tlsconfig.ClientDefault(tlsOpts...)

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
			DialContext:     dialer.DialContext,
		},
		CheckRedirect: client.CheckRedirect,
	}
}
