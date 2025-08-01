package files

import (
	"github.com/apache/dubbo-kubernetes/pkg/filewatcher"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
)

type FileSingleton[T any] struct {
	krt.Singleton[T]
}

func NewFileSingleton[T any](
	fileWatcher filewatcher.FileWatcher,
	filename string,
	readFile func(filename string) (T, error),
	opts ...krt.CollectionOption,
) {
	return
}
