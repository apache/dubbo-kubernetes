/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package files

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/filewatcher"
	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
	"go.uber.org/atomic"
	"time"
)

type FileSingleton[T any] struct {
	krt.Singleton[T]
}

func NewFileSingleton[T any](
	fileWatcher filewatcher.FileWatcher,
	filename string,
	readFile func(filename string) (T, error),
	opts ...krt.CollectionOption,
) (FileSingleton[T], error) {
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
			fmt.Errorf("failed to update: %v", err)
			return
		}
		cur.Store(&cfg)
		trigger.TriggerRecomputation()
	})
	return FileSingleton[T]{sc}, nil
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
