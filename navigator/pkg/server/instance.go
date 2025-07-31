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

package server

import (
	"time"
)

type Component func(stop <-chan struct{}) error

type task struct {
	name string
	task Component
}

type Instance interface {
	Start(stop <-chan struct{}) error
}

type instance struct {
	components chan task
	done       chan struct{}
}

var _ Instance = &instance{}

func New() Instance {
	return &instance{
		done:       make(chan struct{}),
		components: make(chan task, 1000), // should be enough?
	}
}

func (i *instance) Start(stop <-chan struct{}) error {
	shutdown := func() {
		close(i.done)
	}

	for startupDone := false; !startupDone; {
		select {
		case next := <-i.components:
			t0 := time.Now()
			if err := next.task(stop); err != nil {
				// Startup error: terminate and return the error.
				shutdown()
				return err
			}
			runtime := time.Since(t0)
			if runtime > time.Second {
			}
		default:
			startupDone = true
		}
	}

	go func() {
		for {
			select {
			case <-stop:
				shutdown()
				return
			case next := <-i.components:
				t0 := time.Now()
				if err := next.task(stop); err != nil {
				}
				runtime := time.Since(t0)
				if runtime > time.Second {
				}
			}
		}
	}()

	return nil
}
