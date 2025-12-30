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

package ctrlz

import (
	"fmt"
	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"
	"github.com/spf13/cobra"
	"net"
	"net/http"
	"sync"
	"time"
)

var log = dubbolog.RegisterScope("ctrlz", "ctrlz debugging")

const DefaultControlZPort = 9876

type Options struct {
	Port    uint16
	Address string
}

type Server struct {
	listener   net.Listener
	shutdown   sync.WaitGroup
	httpServer http.Server
}

func (s *Server) listen() {
	log.Infof("ControlZ available at %s", s.httpServer.Addr)
	err := s.httpServer.Serve(s.listener)
	log.Infof("ControlZ terminated: %v", err)
	s.shutdown.Done()
}

func (s *Server) Close() {
	log.Info("Closing ControlZ")

	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			log.Warnf("Error closing ControlZ: %v", err)
		}
		s.shutdown.Wait()
	}
}

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
		log.Errorf("Unable to start ControlZ: %v", err)
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

func (o *Options) AttachCobraFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().Uint16Var(&o.Port, "ctrlz_port", o.Port,
		"The IP port to use for the ControlZ introspection facility")
	cmd.PersistentFlags().StringVar(&o.Address, "ctrlz_address", o.Address,
		"The IP Address to listen on for the ControlZ introspection facility. Use '*' to indicate all addresses.")
}
