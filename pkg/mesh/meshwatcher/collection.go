package meshwatcher

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/filewatcher"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	krtfiles "github.com/apache/dubbo-kubernetes/pkg/kube/krt/files"
	"github.com/apache/dubbo-kubernetes/pkg/mesh"
	meshconfig "istio.io/api/mesh/v1alpha1"
	"os"
	"path"
)

type MeshConfigSource = krt.Singleton[string]

type MeshConfigResource struct {
	*meshconfig.MeshConfig
}

func NewFileSource(fileWatcher filewatcher.FileWatcher, filename string, opts krt.OptionsBuilder) (MeshConfigSource, error) {
	return krtfiles.NewFileSingleton[string](fileWatcher, filename, func(filename string) (string, error) {
		b, err := os.ReadFile(filename)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}, opts.WithName("Mesh_File_"+path.Base(filename))...)
}

func NewCollection(opts krt.OptionsBuilder, sources ...MeshConfigSource) krt.Singleton[MeshConfigResource] {
	if len(sources) > 2 {
		panic("currently only 2 sources are supported")
	}
	return krt.NewSingleton[MeshConfigResource](
		func(ctx krt.HandlerContext) *MeshConfigResource {
			meshCfg := mesh.DefaultMeshConfig()

			for _, attempt := range sources {
				s := krt.FetchOne(ctx, attempt.AsCollection())
				if s == nil {
					fmt.Println("mesh configuration source missing")
					continue
				}
				n, err := mesh.ApplyMeshConfig(*s, meshCfg)
				if err != nil {
					if len(sources) == 1 {
						fmt.Errorf("invalid mesh config, using last known state: %v", err)
						ctx.DiscardResult()
						return &MeshConfigResource{mesh.DefaultMeshConfig()}
					}
					fmt.Printf("invalid mesh config, ignoring: %v", err)
					continue
				}
				meshCfg = n
			}
			return &MeshConfigResource{meshCfg}
		}, opts.WithName("MeshConfig")...,
	)
}
