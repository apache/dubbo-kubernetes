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
	"time"
)

import (
	"github.com/go-logr/logr"

	"github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
)

type UpstreamResponse struct {
	ControlPlaneId      string
	Type                model.ResourceType
	AddedResources      model.ResourceList
	RemovedResourcesKey []model.ResourceKey
	IsInitialRequest    bool
}

type Callbacks struct {
	OnResourcesReceived func(upstream UpstreamResponse) error
}

// All methods other than Receive() are non-blocking. It does not wait until the peer CP receives the message.
type DeltaDDSStream interface {
	DeltaDiscoveryRequest(resourceType model.ResourceType) error
	Receive() (UpstreamResponse, error)
	ACK(resourceType model.ResourceType) error
	NACK(resourceType model.ResourceType, err error) error
}

type DDSSyncClient interface {
	Receive() error
}

type ddsSyncClient struct {
	log             logr.Logger
	resourceTypes   []core_model.ResourceType
	callbacks       *Callbacks
	ddsStream       DeltaDDSStream
	responseBackoff time.Duration
}

func NewDDSSyncClient(
	log logr.Logger,
	rt []core_model.ResourceType,
	ddsStream DeltaDDSStream,
	cb *Callbacks,
	responseBackoff time.Duration,
) DDSSyncClient {
	return &ddsSyncClient{
		log:             log,
		resourceTypes:   rt,
		ddsStream:       ddsStream,
		callbacks:       cb,
		responseBackoff: responseBackoff,
	}
}

func (s *ddsSyncClient) Receive() error {
	for _, typ := range s.resourceTypes {
		s.log.V(1).Info("sending DeltaDiscoveryRequest", "type", typ)
		if err := s.ddsStream.DeltaDiscoveryRequest(typ); err != nil {
			return errors.Wrap(err, "discovering failed")
		}
	}

	for {
		received, err := s.ddsStream.Receive()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return errors.Wrap(err, "failed to receive a discoveryengine response")
		}
		s.log.V(1).Info("DeltaDiscoveryResponse received", "response", received)

		if s.callbacks == nil {
			s.log.Info("no callback set, sending ACK", "type", string(received.Type))
			if err := s.ddsStream.ACK(received.Type); err != nil {
				if err == io.EOF {
					return nil
				}
				return errors.Wrap(err, "failed to ACK a discoveryengine response")
			}
			continue
		}
		err = s.callbacks.OnResourcesReceived(received)
		if !received.IsInitialRequest {
			// Execute backoff only on subsequent request.
			// When client first connects, the server sends empty DeltaDiscoveryResponse for every resource type.
			time.Sleep(s.responseBackoff)
		}
		if err != nil {
			s.log.Info("error during callback received, sending NACK", "err", err)
			if err := s.ddsStream.NACK(received.Type, err); err != nil {
				if err == io.EOF {
					return nil
				}
				return errors.Wrap(err, "failed to NACK a discoveryengine response")
			}
		} else {
			s.log.V(1).Info("sending ACK", "type", received.Type)
			if err := s.ddsStream.ACK(received.Type); err != nil {
				if err == io.EOF {
					return nil
				}
				return errors.Wrap(err, "failed to ACK a discoveryengine response")
			}
		}
	}
}
