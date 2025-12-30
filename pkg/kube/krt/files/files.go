//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package files

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/filewatcher"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	"github.com/apache/dubbo-kubernetes/pkg/slices"
	"github.com/apache/dubbo-kubernetes/pkg/util/sets"
	"github.com/fsnotify/fsnotify"
	"go.uber.org/atomic"

	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"
)

const watchDebounceDelay = 50 * time.Millisecond

var log = dubbolog.RegisterScope("krtfiles", "krt files debugging")

var supportedExtensions = sets.New(".yaml", ".yml")

type recursiveWatcher struct {
	*fsnotify.Watcher
}

type FileCollection[T any] struct {
	krt.StaticCollection[T]
	read func() []T
}

type FileSingleton[T any] struct {
	krt.Singleton[T]
}

type FolderWatch[T any] struct {
	root  string
	parse func([]byte) ([]T, error)

	mu        sync.RWMutex
	state     []T
	callbacks []func()
}

func NewFileCollection[F any, O any](w *FolderWatch[F], transform func(F) *O, opts ...krt.CollectionOption) FileCollection[O] {
	res := FileCollection[O]{
		read: func() []O {
			return readSnapshot[F, O](w, transform)
		},
	}
	sc := krt.NewStaticCollection[O](nil, res.read(), opts...)
	w.subscribe(func() {
		now := res.read()
		sc.Reset(now)
	})
	res.StaticCollection = sc
	return res
}

func NewFileSingleton[T any](fileWatcher filewatcher.FileWatcher, filename string, readFile func(filename string) (T, error), opts ...krt.CollectionOption) (FileSingleton[T], error) {
	cfg, err := readFile(filename)
	if err != nil {
		return FileSingleton[T]{}, err
	}

	stop := krt.GetStop(opts...)

	cur := atomic.NewPointer(&cfg)
	trigger := krt.NewRecomputeTrigger(true, opts...)
	sc := krt.NewSingleton[T](func(ctx krt.HandlerContext) *T {
		trigger.MarkDependant(ctx)
		return cur.Load()
	}, opts...)
	sc.AsCollection().WaitUntilSynced(stop)
	watchFile(fileWatcher, filename, stop, func() {
		cfg, err := readFile(filename)
		if err != nil {
			log.Errorf("failed to update: %v", err)
			return
		}
		cur.Store(&cfg)
		trigger.TriggerRecomputation()
	})
	return FileSingleton[T]{sc}, nil
}

func NewFolderWatch[T any](fileDir string, parse func([]byte) ([]T, error), stop <-chan struct{}) (*FolderWatch[T], error) {
	fw := &FolderWatch[T]{root: fileDir, parse: parse}
	// Read initial state
	if err := fw.readOnce(); err != nil {
		return nil, err
	}
	fw.watch(stop)
	return fw, nil
}

func (f *FolderWatch[T]) readOnce() error {
	var result []T

	err := filepath.Walk(f.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		} else if !supportedExtensions.Contains(filepath.Ext(path)) || (info.Mode()&os.ModeType) != 0 {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			log.Errorf("Failed to readOnce %s: %v", path, err)
			return err
		}
		parsed, err := f.parse(data)
		if err != nil {
			log.Errorf("Failed to parse %s: %v", path, err)
			return err
		}
		result = append(result, parsed...)
		return nil
	})
	if err != nil {
		log.Errorf("failure during filepath.Walk: %v", err)
	}

	if err != nil {
		return err
	}

	f.mu.Lock()
	f.state = result
	cb := slices.Clone(f.callbacks)
	f.mu.Unlock()
	for _, c := range cb {
		c()
	}
	return nil
}

func (f *FolderWatch[T]) watch(stop <-chan struct{}) {
	c := make(chan struct{}, 1)
	if err := f.fileTrigger(c, stop); err != nil {
		log.Errorf("Unable to setup FileTrigger for %s: %v", f.root, err)
		return
	}
	// Run the close loop asynchronously.
	go func() {
		for {
			select {
			case <-c:
				log.Infof("Triggering reload of file configuration")
				if err := f.readOnce(); err != nil {
					log.Errorf("unable to reload file configuration %v: %v", f.root, err)
				}
			case <-stop:
				return
			}
		}
	}()
}

func (f *FolderWatch[T]) get() []T {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state
}

func (f *FolderWatch[T]) subscribe(fn func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.callbacks = append(f.callbacks, fn)
}

func (f *FolderWatch[T]) fileTrigger(events chan struct{}, stop <-chan struct{}) error {
	fs, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	watcher := recursiveWatcher{fs}
	if err = watcher.watchRecursive(f.root); err != nil {
		return err
	}
	go func() {
		defer watcher.Close()
		var debounceC <-chan time.Time
		for {
			select {
			case <-debounceC:
				debounceC = nil
				events <- struct{}{}
			case e := <-watcher.Events:
				s, err := os.Stat(e.Name)
				if err == nil && s != nil && s.IsDir() {
					// If it's a directory, add a watch for it so we see nested files.
					if e.Op&fsnotify.Create != 0 {
						log.Debugf("add watch for %v: %v", s.Name(), watcher.watchRecursive(e.Name))
					}
				}
				// Can't stat a deleted directory, so attempt to remove it. If it fails it is not a problem
				if e.Op&fsnotify.Remove != 0 {
					_ = watcher.Remove(e.Name)
				}
				if debounceC == nil {
					debounceC = time.After(watchDebounceDelay)
				}
			case err := <-watcher.Errors:
				log.Errorf("Error watching file trigger: %v %v", f.root, err)
				return
			case <-stop:
				log.Infof("Shutting down file watcher: %v", f.root)
				return
			}
		}
	}()
	return nil
}

func (m recursiveWatcher) watchRecursive(path string) error {
	err := filepath.Walk(path, func(walkPath string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			if err = m.Watcher.Add(walkPath); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func watchFile(fileWatcher filewatcher.FileWatcher, file string, stop <-chan struct{}, callback func()) {
	_ = fileWatcher.Add(file)
	go func() {
		var timerC <-chan time.Time
		for {
			select {
			case <-stop:
				return
			case <-timerC:
				timerC = nil
				callback()
			case <-fileWatcher.Events(file):
				if timerC == nil {
					timerC = time.After(100 * time.Millisecond)
				}
			}
		}
	}()
}

func readSnapshot[F any, O any](w *FolderWatch[F], transform func(F) *O) []O {
	res := w.get()
	return slices.MapFilter(res, transform)
}
