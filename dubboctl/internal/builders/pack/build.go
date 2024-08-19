// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pack

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"
	"time"
)

import (
	"github.com/buildpacks/pack/pkg/buildpack"
	packClient "github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/logging"
	"github.com/buildpacks/pack/pkg/project/types"

	"github.com/distribution/reference"

	"github.com/docker/docker/client"

	"github.com/heroku/color"
)

import (
	"github.com/apache/dubbo-kubernetes/dubboctl/internal/builders"
	"github.com/apache/dubbo-kubernetes/dubboctl/internal/builders/pack/mirror"
	"github.com/apache/dubbo-kubernetes/dubboctl/internal/docker"
	"github.com/apache/dubbo-kubernetes/dubboctl/internal/dubbo"
)

// DefaultName when no WithName option is provided to NewBuilder
const DefaultName = builders.Pack

var (
	DefaultBaseBuilder = "docker.io/paketobuildpacks/builder-jammy-tiny:latest"
	DefaultTinyBuilder = "docker.io/paketobuildpacks/builder-jammy-tiny:latest"
)

var (
	DefaultBuilderImages = map[string]string{
		"go":   DefaultTinyBuilder,
		"java": DefaultBaseBuilder,
	}

	// Ensure that all entries in this list are terminated with a trailing "/"
	// See GHSA-5336-2g3f-9g3m for details
	trustedBuilderImagePrefixes = []string{
		"quay.io/boson/",
		"gcr.io/paketo-buildpacks/",
		"docker.io/paketobuildpacks/",
		"ghcr.io/vmware-tanzu/function-buildpacks-for-knative/",
		"gcr.io/buildpacks/",
		"ghcr.io/knative/",
	}

	defaultBuildpacks = map[string][]string{
		"go":   {"paketo-buildpacks/go"},
		"java": {"paketo-buildpacks/java"},
	}

	// work around for resolving the images with latest tag could not be mirrored
	latestImageTagMapping = map[string]string{
		"paketobuildpacks/builder-jammy-tiny": "0.0.271",
		"paketobuildpacks/run-jammy-tiny":     "0.2.46",
		"buildpacksio/lifecycle":              "0.17.7",
	}
)

// Builder will build Function using Pack.
type Builder struct {
	name string
	// in non-verbose mode contains std[err,out], so it can be printed on error
	outBuff       bytes.Buffer
	logger        logging.Logger
	impl          Impl
	withTimestamp bool
}

// Impl allows for the underlying implementation to be mocked for tests.
type Impl interface {
	Build(context.Context, packClient.BuildOptions) error
}

// NewBuilder instantiates a Buildpack-based Builder
func NewBuilder(options ...Option) *Builder {
	b := &Builder{name: DefaultName}
	for _, o := range options {
		o(b)
	}
	// Stream logs to stdout or buffer only for display on error.
	b.logger = logging.NewLogWithWriters(color.Stdout(), color.Stderr(), logging.WithVerbose())

	return b
}

type Option func(*Builder)

func WithName(n string) Option {
	return func(b *Builder) {
		b.name = n
	}
}

func WithImpl(i Impl) Option {
	return func(b *Builder) {
		b.impl = i
	}
}

func transportEnv(ee []dubbo.Env) (map[string]string, error) {
	envs := make(map[string]string, len(ee))
	for _, e := range ee {
		// Assert non-nil name.
		if e.Name == nil {
			return envs, errors.New("env name may not be nil")
		}
		// Nil value indicates the resultant map should not include this env var.
		if e.Value == nil {
			continue
		}
		k, v := *e.Name, *e.Value
		envs[k] = v
	}
	return envs, nil
}

// work around for resolving the images with latest tag could not be mirrored
func defaultBuildPackImageReplace(image string) string {
	ref, err := reference.ParseDockerRef(image)
	if err != nil {
		return image
	}
	if reference.Domain(ref) != "docker.io" {
		return image
	}
	if tag, ok := ref.(reference.Tagged); ok && tag.Tag() == "latest" {
		image = fmt.Sprintf("%s/%s:%s", reference.Domain(ref), reference.Path(ref), latestImageTagMapping[reference.Path(ref)])
		return image
	}
	return image
}

// Build the Function at path.
func (b *Builder) Build(ctx context.Context, f *dubbo.Dubbo) (err error) {
	// Builder image from the function if defined, default otherwise.
	image, err := BuilderImage(f, b.name)
	if err != nil {
		return
	}

	buildpacks := f.Build.Buildpacks
	if len(buildpacks) == 0 {
		// check if the default builder image is used
		ref, err := reference.ParseDockerRef(image)
		if err == nil &&
			reference.Domain(ref) == "docker.io" &&
			reference.Path(ref) == "paketobuildpacks/builder-jammy-tiny" {
			// use the default buildpacks for the runtime
			buildpacks = defaultBuildpacks[f.Runtime]
		}
	}

	// Pack build options
	opts := packClient.BuildOptions{
		AppPath:    f.Root,
		Image:      f.Image,
		Builder:    image,
		Buildpacks: buildpacks,
		ProjectDescriptor: types.Descriptor{
			Build: types.Build{
				Exclude: []string{},
			},
		},
		ContainerConfig: packClient.ContainerConfig{Network: "", Volumes: nil},
		// use the default user/group
		// 0 means root (which may cause permission issues when running the builder with non-root user)
		GroupID: -1,
		UserID:  -1,
	}
	if b.withTimestamp {
		now := time.Now()
		opts.CreationTime = &now
	}
	opts.Env, err = transportEnv(f.Build.BuildEnvs)
	if err != nil {
		return err
	}
	if runtime.GOOS == "linux" {
		opts.ContainerConfig.Network = "host"
	}

	// TODO add it or not ?
	// only trust our known builders
	// opts.TrustBuilder = TrustBuilder

	impl := b.impl
	// Instantiate the pack build client implementation
	// (and update build opts as necessary)
	if impl == nil {
		var (
			cli        client.CommonAPIClient
			dockerHost string
		)

		cli, dockerHost, err = docker.NewClient(client.DefaultDockerHost)
		if err != nil {
			return fmt.Errorf("cannot create docker client: %w", err)
		}
		defer cli.Close()
		opts.DockerHost = dockerHost

		var fetcher buildpack.ImageFetcher = nil
		if f.Build.CnMirror {
			fetcher = mirror.NewMirrorFetcher(b.logger, cli, defaultBuildPackImageReplace)
		}

		// Client with a logger which is enabled if in Verbose mode and a dockerClient that supports SSH docker daemon connection.
		if impl, err = packClient.NewClient(packClient.WithLogger(b.logger), packClient.WithDockerClient(cli), packClient.WithFetcher(fetcher)); err != nil {
			return fmt.Errorf("cannot create pack client: %w", err)
		}
	}

	// Perform the build
	if err = impl.Build(ctx, opts); err != nil {
		if ctx.Err() != nil {
			return // SIGINT
		} else {
			err = fmt.Errorf("failed to build the application: %w", err)
			fmt.Fprintln(color.Stderr(), "")
			_, _ = io.Copy(color.Stderr(), &b.outBuff)
			fmt.Fprintln(color.Stderr(), "")
		}
	}
	return
}

// TrustBuilder determines whether the builder image should be trusted
// based on a set of trusted builder image registry prefixes.
func TrustBuilder(b string) bool {
	for _, v := range trustedBuilderImagePrefixes {
		// Ensure that all entries in this list are terminated with a trailing "/"
		if !strings.HasSuffix(v, "/") {
			v = v + "/"
		}
		if strings.HasPrefix(b, v) {
			return true
		}
	}
	return false
}

// BuilderImage Image chooses the correct builder image or defaults.
func BuilderImage(f *dubbo.Dubbo, builderName string) (string, error) {
	return builders.Image(f, builderName, DefaultBuilderImages)
}

// Errors

type ErrRuntimeRequired struct{}

func (e ErrRuntimeRequired) Error() string {
	return "Pack requires the Function define a language runtime"
}

type ErrRuntimeNotSupported struct {
	Runtime string
}

func (e ErrRuntimeNotSupported) Error() string {
	return fmt.Sprintf("Pack builder has no default builder image for the '%v' language runtime.  Please provide one.", e.Runtime)
}
