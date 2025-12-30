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

package pixiu

import (
	"context"
	"errors"
	"time"

	"go.uber.org/atomic"
)

var errAbort = errors.New("gateway aborted")

// Agent manages Pixiu proxy lifecycle
type Agent struct {
	proxy Proxy

	statusCh chan exitStatus
	abortCh  chan error

	terminationDrainDuration time.Duration
	minDrainDuration         time.Duration

	skipDrain *atomic.Bool
}

type exitStatus struct {
	err error
}

// NewAgent creates a new proxy agent for Pixiu start-up and clean-up functions
func NewAgent(proxy Proxy, terminationDrainDuration, minDrainDuration time.Duration) *Agent {
	return &Agent{
		proxy:                    proxy,
		statusCh:                 make(chan exitStatus, 1),
		abortCh:                  make(chan error, 1),
		terminationDrainDuration: terminationDrainDuration,
		minDrainDuration:         minDrainDuration,
		skipDrain:                atomic.NewBool(false),
	}
}

// Run starts the Pixiu and waits until it terminates
func (a *Agent) Run(ctx context.Context) {
	log.Info("starting gateway proxy agent")
	go a.runWait(a.abortCh)

	select {
	case status := <-a.statusCh:
		if status.err != nil {
			log.Errorf("gateway exited with error: %v", status.err)
		} else {
			log.Infof("gateway exited normally")
		}

	case <-ctx.Done():
		a.terminate()
		log.Info("agent has successfully terminated")
	}
}

func (a *Agent) DisableDraining() {
	a.skipDrain.Store(true)
}

func (a *Agent) DrainNow() {
	log.Infof("agent draining Pixiu")
	err := a.proxy.Drain(true)
	if err != nil {
		log.Warnf("error in invoking drain: %v", err)
	}
	a.DisableDraining()
}

// terminate starts exiting the process
func (a *Agent) terminate() {
	log.Infof("agent draining gateway proxy for termination")
	if a.skipDrain.Load() {
		log.Infof("agent already drained, exiting immediately")
		a.abortCh <- errAbort
		return
	}
	e := a.proxy.Drain(false)
	if e != nil {
		log.Warnf("error in invoking drain: %v", e)
	}

	log.Infof("graceful termination period is %v, starting...", a.terminationDrainDuration)
	select {
	case status := <-a.statusCh:
		log.Infof("pixiu exited with status %v", status.err)
		log.Infof("graceful termination logic ended prematurely, gateway process terminated early")
		return
	case <-time.After(a.terminationDrainDuration):
		log.Infof("graceful termination period complete, terminating remaining processes.")
		a.abortCh <- errAbort
	}
	status := <-a.statusCh
	if status.err == errAbort {
		log.Infof("gateway aborted normally")
	} else {
		log.Warnf("gateway aborted abnormally")
	}
	log.Warnf("aborted gateway instance")
}

// runWait runs the start-up command as a go routine and waits for it to finish
func (a *Agent) runWait(abortCh <-chan error) {
	err := a.proxy.Run(abortCh)
	a.proxy.Cleanup()
	a.statusCh <- exitStatus{err: err}
}
