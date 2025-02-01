package ssh

import (
	"context"
	"golang.org/x/crypto/ssh"
	"net"
)

type (
	PasswordCallback   func() (string, error)
	PassPhraseCallback func() (string, error)
	HostKeyCallback    func(hostPort string, pubKey ssh.PublicKey) error
)

type Config struct {
	Identity           string
	PassPhrase         string
	PasswordCallback   PasswordCallback
	PassPhraseCallback PassPhraseCallback
	HostKeyCallback    HostKeyCallback
}

type dialer struct {
	sshClient *ssh.Client
	network   string
	addr      string
}

type ContextDialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}
type DialContextFn = func(ctx context.Context, network, addr string) (net.Conn, error)

type contextDialerFn DialContextFn

func (n contextDialerFn) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return n(ctx, network, address)
}
