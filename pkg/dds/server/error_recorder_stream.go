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
	envoy_sd "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/stream/v3"
	"io"
	"sync"
)

// ErrorRecorderStream is a DeltaStream that records an error
// We need this because go-control-plane@v0.11.1/pkg/server/delta/v3/server.go:190 swallows an error on Recv()
type ErrorRecorderStream interface {
	stream.DeltaStream
	Err() error
}

type errorRecorderStream struct {
	stream.DeltaStream
	err error
	sync.Mutex
}

var _ stream.DeltaStream = &errorRecorderStream{}

func NewErrorRecorderStream(s stream.DeltaStream) ErrorRecorderStream {
	return &errorRecorderStream{
		DeltaStream: s,
	}
}

func (e *errorRecorderStream) Recv() (*envoy_sd.DeltaDiscoveryRequest, error) {
	res, err := e.DeltaStream.Recv()
	if err != nil && err != io.EOF { // do not consider "end of stream" an error
		e.Lock()
		e.err = err
		e.Unlock()
	}
	return res, err
}

func (e *errorRecorderStream) Err() error {
	e.Lock()
	defer e.Unlock()
	return e.err
}
