package builder

import (
	"context"
	"github.com/apache/dubbo-kubernetes/dubboctl/pkg/sdk/dubbo"
)

type Builder struct{}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b Builder) Build(ctx context.Context, f *dubbo.DubboConfig) error {
	cli, _, err := docker.NewClient(client.DefaultDockerHost)
	if err != nil {
		return err
	}
	buildOpts := types.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{f.Image},
	}

	buildCtx, _ := archive.TarWithOptions(f.Root, &archive.TarOptions{})
	resp, err := cli.ImageBuild(ctx, buildCtx, buildOpts)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	termFd, isTerm := term.GetFdInfo(os.Stderr)
	err = jsonmessage.DisplayJSONMessagesStream(resp.Body, os.Stderr, termFd, isTerm, nil)
	if err != nil {
		return err
	}
	return nil
}
