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
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
)

type Callbacks struct {
	OnMappingSyncRequestReceived  func(request *mesh_proto.MappingSyncRequest) error
	OnMetadataSyncRequestReceived func(request *mesh_proto.MetadataSyncRequest) error
}

// DubboSyncClient Handle Dubbo Sync Request from client
type DubboSyncClient interface {
	ClientID() string
	HandleReceive() error
	Send(resourceList core_model.ResourceList, revision int64) error
}

type dubboSyncClient struct {
	log        logr.Logger
	id         string
	syncStream DubboSyncStream
	callbacks  *Callbacks
}

func NewDubboSyncClient(log logr.Logger, id string, syncStream DubboSyncStream, cb *Callbacks) DubboSyncClient {
	return &dubboSyncClient{
		log:        log,
		id:         id,
		syncStream: syncStream,
		callbacks:  cb,
	}
}

func (s *dubboSyncClient) ClientID() string {
	return s.id
}

func (s *dubboSyncClient) HandleReceive() error {
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
		switch received.(type) {
		case *mesh_proto.MappingSyncRequest:
			err = s.callbacks.OnMappingSyncRequestReceived(received.(*mesh_proto.MappingSyncRequest))
			if err != nil {
				s.log.Error(err, "error in OnMappingSyncRequestReceived")
			} else {
				s.log.Info("OnMappingSyncRequestReceived successed")
			}
		case *mesh_proto.MetadataSyncRequest:
			panic("unimplemented")
		default:
			return errors.New("unknown type request")
		}
	}
}

func (s *dubboSyncClient) Send(resourceList core_model.ResourceList, revision int64) error {
	return s.syncStream.Send(resourceList, revision)
}
