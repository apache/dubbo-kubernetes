package kube

type client struct {
}

type Client interface {
}

type CliClient interface {
}

var (
	_ Client    = &client{}
	_ CliClient = &client{}
)