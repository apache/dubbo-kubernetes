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

package mux

import (
	"context"
	"errors"
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"io"
	"sync"
	"time"
)

type Session interface {
	ServerStream() mesh_proto.DubboDiscoveryService_StreamDubboResourcesServer
	ClientStream() mesh_proto.DubboDiscoveryService_StreamDubboResourcesClient
	PeerID() string
	Error() <-chan error
	SetError(err error)
}

type session struct {
	peerID       string
	serverStream *ddsServerStream
	clientStream *ddsClientStream

	err       chan error
	sync.Once // protects err, so we only send the first error and close the channel
}

// handleRecv polls to receive messages from the DDSStream (the actual grpc bidi-stream).
// Depending on the message it dispatches to either the server receive buffer or the client receive buffer.
// It also closes both streams when an error on the recv side happens.
// We can rely on an error on recv to end the session because we're sure an error on recv will always happen, it might be io.EOF if we're just done.
func (s *session) handleRecv(stream MultiplexStream) {
	msg, err := stream.Recv()
	if err != nil {
		s.clientStream.bufferStream.close()
		s.serverStream.bufferStream.close()
		// Recv always finishes with either an EOF or another error
		s.SetError(err)
		return
	}
	switch v := msg.Value.(type) {
	case *mesh_proto.Message_LegacyRequest:
		msg = &mesh_proto.Message{Value: &mesh_proto.Message_Request{Request: DiscoveryRequestV3(v.LegacyRequest)}}
	case *mesh_proto.Message_LegacyResponse:
		msg = &mesh_proto.Message{Value: &mesh_proto.Message_Response{Response: DiscoveryResponseV3(v.LegacyResponse)}}
	}
	// We can safely not care about locking as we're only closing the channel from this goroutine.
	switch msg.Value.(type) {
	case *mesh_proto.Message_Request:
		s.serverStream.bufferStream.recvBuffer <- msg
	case *mesh_proto.Message_Response:
		s.clientStream.bufferStream.recvBuffer <- msg
	}
}

// handleSend polls either sendBuffer and call send on the DDSStream (the actual grpc bidi-stream).
// This call is stopped whenever either of the sendBuffer are closed (in practice they are always closed together anyway).
func (s *session) handleSend(stream MultiplexStream, sendTimeout time.Duration) {
	for {
		var msgToSend *mesh_proto.Message
		select {
		case msg, more := <-s.serverStream.bufferStream.sendBuffer:
			if !more {
				return
			}
			msgToSend = msg
		case msg, more := <-s.clientStream.bufferStream.sendBuffer:
			if !more {
				return
			}
			msgToSend = msg
		}
		ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
		go func() {
			<-ctx.Done()
			if ctx.Err() == context.DeadlineExceeded {
				// This is very unlikely to happen, but it was introduced as a last resort protection from a gRPC streaming deadlock.
				// gRPC streaming deadlock may happen if both peers are stuck on Send() operation without calling Recv() often enough.
				// In this case, if data is big enough, both parties may wait for WINDOW_UPDATE on HTTP/2 stream.
				// We fixed the deadlock by increasing buffer size which is larger that all possible inflight request.
				// If the connection is broken and send is stuck, it's more likely for gRPC keep alive to catch such case.
				// If you still hit the timeout without deadlock, you may increase it. However, there are two possible scenarios
				// 1) This is a malicious client reading stream byte by byte. In this case it's actually better to end the stream
				// 2) A client is such overwhelmed that it cannot even let the server know that it's ready to receive more data.
				//    In this case it's recommended to scale number of instances.
				s.SetError(errors.New("timeout while sending a message to peer"))
			}
		}()
		if err := stream.Send(msgToSend); err != nil {
			s.SetError(err)
			cancel()
			return
		}
		cancel()
	}
}

type MultiplexStream interface {
	Send(message *mesh_proto.Message) error
	Recv() (*mesh_proto.Message, error)
	Context() context.Context
}

type bufferStream struct {
	sendBuffer chan *mesh_proto.Message
	recvBuffer chan *mesh_proto.Message

	// Protects the send-buffer against writing on a closed channel, this is needed as we don't control in which goroutine `Send` will be called.
	lock   sync.Mutex
	closed bool
}

func (s *session) SetError(err error) {
	// execute this once so writers to this channel won't be stuck or trying to write to a close channel
	// We only care about the first error, because it results in broken session anyway.
	s.Once.Do(func() {
		s.err <- err
		close(s.err)
	})
}

func (s *session) ServerStream() mesh_proto.DubboDiscoveryService_StreamDubboResourcesServer {
	return s.serverStream
}

func (s *session) ClientStream() mesh_proto.DubboDiscoveryService_StreamDubboResourcesClient {
	return s.clientStream
}

func (s *session) PeerID() string {
	return s.peerID
}

func (s *session) Error() <-chan error {
	return s.err
}

func newBufferStream(bufferSize uint32) *bufferStream {
	return &bufferStream{
		sendBuffer: make(chan *mesh_proto.Message, bufferSize),
		recvBuffer: make(chan *mesh_proto.Message, bufferSize),
	}
}

func (k *bufferStream) Send(message *mesh_proto.Message) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	if k.closed {
		return io.EOF
	}
	k.sendBuffer <- message
	return nil
}

func (k *bufferStream) Recv() (*mesh_proto.Message, error) {
	r, more := <-k.recvBuffer
	if !more {
		return nil, io.EOF
	}
	return r, nil
}

func (k *bufferStream) close() {
	k.lock.Lock()
	defer k.lock.Unlock()

	k.closed = true
	close(k.sendBuffer)
	close(k.recvBuffer)
}
