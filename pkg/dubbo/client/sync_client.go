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

package client

import (
	"io"
)

import (
	"github.com/go-logr/logr"

	"github.com/pkg/errors"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	core_mesh "github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/mesh"
)

type Callbacks struct {
	OnRequestReceived func(request *mesh_proto.MappingSyncRequest) error
}

// MappingSyncClient Handle MappingSyncRequest from client
type MappingSyncClient interface {
	ClientID() string
	HandleReceive() error
	Send(mappingList *core_mesh.MappingResourceList, revision int64) error
}

type mappingSyncClient struct {
	log        logr.Logger
	id         string
	syncStream MappingSyncStream
	callbacks  *Callbacks
}

func NewMappingSyncClient(log logr.Logger, id string, syncStream MappingSyncStream, cb *Callbacks) MappingSyncClient {
	return &mappingSyncClient{
		log:        log,
		id:         id,
		syncStream: syncStream,
		callbacks:  cb,
	}
}

func (s *mappingSyncClient) ClientID() string {
	return s.id
}

func (s *mappingSyncClient) HandleReceive() error {
	for {
		received, err := s.syncStream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return errors.Wrap(err, "failed to receive a MappingSyncRequest")
		}

		if s.callbacks == nil {
			// if no callbacks
			s.log.Info("no callback set")
			continue
		}

		// callbacks
		err = s.callbacks.OnRequestReceived(received)
		if err != nil {
			s.log.Error(err, "error in OnRequestReceived")
		} else {
			s.log.Info("OnRequestReceived successed")
		}

	}
}

func (s *mappingSyncClient) Send(mappingList *core_mesh.MappingResourceList, revision int64) error {
	return s.syncStream.Send(mappingList, revision)
}
