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
)

var errAbort = errors.New("gateway aborted")

// Agent manages Pixiu proxy lifecycle
type Agent struct {
	proxy Proxy

	statusCh chan exitStatus
	abortCh  chan error

	terminationDrainDuration time.Duration
	minDrainDuration         time.Duration
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
	}
}

func (a *Agent) Run(ctx context.Context) {
	pixiulog.Info("Starting gateway agent")
	go a.runWait(a.abortCh)

	select {
	case status := <-a.statusCh:
		if status.err != nil {
			pixiulog.Errorf("gateway exited with error: %v", status.err)
		} else {
			pixiulog.Infof("gateway exited normally")
		}

	case <-ctx.Done():
		a.terminate()
		pixiulog.Info("agent has successfully terminated")
	}
}

// terminate starts exiting the process
func (a *Agent) terminate() {
	pixiulog.Infof("agent draining gateway proxy for termination")
	pixiulog.Infof("graceful termination period is %v, starting...", a.terminationDrainDuration)
	select {
	case status := <-a.statusCh:
		pixiulog.Infof("pixiu exited with status %v", status.err)
		pixiulog.Infof("graceful termination logic ended prematurely, gateway process terminated early")
		return
	case <-time.After(a.terminationDrainDuration):
		pixiulog.Infof("graceful termination period complete, terminating remaining processes.")
		a.abortCh <- errAbort
	}
	status := <-a.statusCh
	if status.err == errAbort {
		pixiulog.Infof("gateway aborted normally")
	} else {
		pixiulog.Warnf("gateway aborted abnormally")
	}
	pixiulog.Warnf("aborted gateway instance")
}

// runWait runs the start-up command as a go routine and waits for it to finish
func (a *Agent) runWait(abortCh <-chan error) {
	err := a.proxy.Run(abortCh)
	a.proxy.Cleanup()
	a.statusCh <- exitStatus{err: err}
}
