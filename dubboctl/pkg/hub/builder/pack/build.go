package pack

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/hub"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/hub/builder"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk/dubbo"
	pack "github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/logging"
	"github.com/buildpacks/pack/pkg/project/types"
	"github.com/docker/docker/client"
	"github.com/heroku/color"
	"io"
	"runtime"
	"strings"
	"time"
)

const DefaultName = builder.Pack

var (
	DefaultBaseBuilder = "ghcr.io/knative/builder-jammy-base:latest"
	DefaultTinyBuilder = "ghcr.io/knative/builder-jammy-tiny:latest"
)

var (
	DefaultBuilderImages = map[string]string{
		"go":   DefaultTinyBuilder,
		"java": DefaultBaseBuilder,
	}

	trustedBuilderImagePrefixes = []string{
		"quay.io/boson/",
		"gcr.io/paketo-buildpacks/",
		"docker.io/paketobuildpacks/",
		"ghcr.io/vmware-tanzu/function-buildpacks-for-knative/",
		"gcr.io/buildpacks/",
		"ghcr.io/knative/",
	}

	defaultBuildpacks = map[string][]string{}
)

type Builder struct {
	name          string
	outBuff       bytes.Buffer
	logger        logging.Logger
	impl          Impl
	withTimestamp bool
}

type Impl interface {
	Build(context.Context, pack.BuildOptions) error
}

type Option func(*Builder)

func NewBuilder(options ...Option) *Builder {
	b := &Builder{name: DefaultName}
	for _, o := range options {
		o(b)
	}
	b.logger = logging.NewLogWithWriters(color.Stdout(), color.Stderr(), logging.WithVerbose())

	return b
}

func BuilderImage(dc *dubbo.DubboConfig, builderName string) (string, error) {
	return builder.Image(dc, builderName, DefaultBuilderImages)
}

func (b *Builder) Build(ctx context.Context, dc *dubbo.DubboConfig) (err error) {
	image, err := BuilderImage(dc, b.name)
	if err != nil {
		return
	}

	buildpacks := dc.Build.Buildpacks
	if len(buildpacks) == 0 {
		buildpacks = defaultBuildpacks[dc.Runtime]
	}

	opts := pack.BuildOptions{
		AppPath:    dc.Root,
		Image:      dc.Image,
		Builder:    image,
		Buildpacks: buildpacks,
		ProjectDescriptor: types.Descriptor{
			Build: types.Build{
				Exclude: []string{},
			},
		},
		ContainerConfig: struct {
			Network string
			Volumes []string
		}{Network: "", Volumes: nil},
	}
	if b.withTimestamp {
		now := time.Now()
		opts.CreationTime = &now
	}
	opts.Env, err = transportEnv(dc.Build.BuildEnvs)
	if err != nil {
		return err
	}
	if runtime.GOOS == "linux" {
		opts.ContainerConfig.Network = "host"
	}

	impl := b.impl
	if impl == nil {
		var (
			cli        client.CommonAPIClient
			dockerHost string
		)

		cli, dockerHost, err = hub.NewClient(client.DefaultDockerHost)
		if err != nil {
			return fmt.Errorf("cannot create docker client: %w", err)
		}
		defer cli.Close()
		opts.DockerHost = dockerHost

		if impl, err = pack.NewClient(pack.WithLogger(b.logger), pack.WithDockerClient(cli)); err != nil {
			return fmt.Errorf("cannot create pack client: %w", err)
		}
	}

	if err = impl.Build(ctx, opts); err != nil {
		if ctx.Err() != nil {
			return
		} else {
			err = fmt.Errorf("failed to build the application: %w", err)
			fmt.Fprintln(color.Stderr(), "")
			_, _ = io.Copy(color.Stderr(), &b.outBuff)
			fmt.Fprintln(color.Stderr(), "")
		}
	}
	return
}

func transportEnv(ee []dubbo.Env) (map[string]string, error) {
	envs := make(map[string]string, len(ee))
	for _, e := range ee {
		// Assert non-nil name.
		if e.Name == nil {
			return envs, errors.New("env name may not be nil")
		}
		if e.Value == nil {
			continue
		}
		k, v := *e.Name, *e.Value
		envs[k] = v
	}
	return envs, nil
}

func TrustBuilder(b string) bool {
	for _, v := range trustedBuilderImagePrefixes {
		if !strings.HasSuffix(v, "/") {
			v = v + "/"
		}
		if strings.HasPrefix(b, v) {
			return true
		}
	}
	return false
}
