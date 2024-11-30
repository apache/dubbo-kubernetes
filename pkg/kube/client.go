package kube

type client struct {
}

type Client interface {
}

type CLIClient interface {
}

var (
	_ Client    = &client{}
	_ CLIClient = &client{}
)
