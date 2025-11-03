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

package xds

import (
	"sync"

	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
)

type PushQueue struct {
	cond         *sync.Cond
	pending      map[*Connection]*model.PushRequest
	queue        []*Connection
	processing   map[*Connection]*model.PushRequest
	shuttingDown bool
}

func NewPushQueue() *PushQueue {
	return &PushQueue{
		pending:    make(map[*Connection]*model.PushRequest),
		processing: make(map[*Connection]*model.PushRequest),
		cond:       sync.NewCond(&sync.Mutex{}),
	}
}

func (p *PushQueue) Enqueue(con *Connection, pushRequest *model.PushRequest) {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	if p.shuttingDown {
		return
	}

	if request, f := p.processing[con]; f {
		p.processing[con] = request.CopyMerge(pushRequest)
		return
	}

	if request, f := p.pending[con]; f {
		p.pending[con] = request.CopyMerge(pushRequest)
		return
	}

	p.pending[con] = pushRequest
	p.queue = append(p.queue, con)
	p.cond.Signal()
}

func (p *PushQueue) Dequeue() (con *Connection, request *model.PushRequest, shutdown bool) {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	for len(p.queue) == 0 && !p.shuttingDown {
		p.cond.Wait()
	}

	if len(p.queue) == 0 {
		// We must be shutting down.
		return nil, nil, true
	}

	con = p.queue[0]
	p.queue[0] = nil
	p.queue = p.queue[1:]

	request = p.pending[con]
	delete(p.pending, con)

	p.processing[con] = nil

	return con, request, false
}

func (p *PushQueue) MarkDone(con *Connection) {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	request := p.processing[con]
	delete(p.processing, con)

	if request != nil {
		p.pending[con] = request
		p.queue = append(p.queue, con)
		p.cond.Signal()
	}
}

func (p *PushQueue) Pending() int {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	return len(p.queue)
}

func (p *PushQueue) ShutDown() {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	p.shuttingDown = true
	p.cond.Broadcast()
}
