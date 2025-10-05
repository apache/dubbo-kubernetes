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

package ctrlz

import (
	"fmt"
	"k8s.io/klog/v2"
	"net"
	"net/http"
	"sync"
	"time"
)

type Options struct {
	Port    uint16
	Address string
}

type Server struct {
	listener   net.Listener
	shutdown   sync.WaitGroup
	httpServer http.Server
}

const DefaultControlZPort = 9876

func DefaultOptions() *Options {
	return &Options{
		Port:    DefaultControlZPort,
		Address: "localhost",
	}
}

func Run(o *Options) (*Server, error) {
	addr := o.Address
	if addr == "*" {
		addr = ""
	}

	// Canonicalize the address and resolve a dynamic port if necessary
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", addr, o.Port))
	if err != nil {
		klog.Errorf("Unable to start ControlZ: %v", err)
		return nil, err
	}

	s := &Server{
		listener: listener,
		httpServer: http.Server{
			Addr:         listener.Addr().(*net.TCPAddr).String(),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: time.Minute, // High timeout to allow profiles to run
		},
	}

	s.shutdown.Add(1)
	go s.listen()

	return s, nil
}

func (s *Server) listen() {
	klog.Infof("ControlZ available at %s", s.httpServer.Addr)
	err := s.httpServer.Serve(s.listener)
	klog.Infof("ControlZ terminated: %v", err)
	s.shutdown.Done()
}

func (s *Server) Close() {
	klog.Info("Closing ControlZ")

	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			klog.Warningf("Error closing ControlZ: %v", err)
		}
		s.shutdown.Wait()
	}
}
