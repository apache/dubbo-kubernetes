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

package introspection

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"
	"github.com/spf13/cobra"
)

var log = dubbolog.RegisterScope("introspection", "process introspection listener")

const (
	DefaultPort = 9876

	PortFlagName    = "diag_port"
	AddressFlagName = "diag_address"

	portFlagHelp    = "The IP port to use for the process introspection listener"
	addressFlagHelp = "The IP address to listen on for the process introspection listener. Use '*' to indicate all addresses."
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

func (s *Server) listen() {
	log.Infof("introspection available at %s", s.httpServer.Addr)
	err := s.httpServer.Serve(s.listener)
	log.Infof("introspection terminated: %v", err)
	s.shutdown.Done()
}

func (s *Server) Close() {
	log.Info("closing introspection")

	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			log.Warnf("error closing introspection: %v", err)
		}
		s.shutdown.Wait()
	}
}

func DefaultOptions() *Options {
	return &Options{
		Port:    DefaultPort,
		Address: "localhost",
	}
}

// ListenAddress is host:port the introspection listener binds, including
// the wildcard-host form produced when Address is "*".
func (o *Options) ListenAddress() string {
	if o == nil {
		return ""
	}
	addr := o.Address
	if addr == "*" {
		addr = ""
	}
	return net.JoinHostPort(addr, strconv.Itoa(int(o.Port)))
}

func Run(o *Options) (*Server, error) {
	listener, err := net.Listen("tcp", o.ListenAddress())
	if err != nil {
		log.Errorf("unable to start introspection: %v", err)
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
	cmd.PersistentFlags().Uint16Var(&o.Port, PortFlagName, o.Port, portFlagHelp)
	cmd.PersistentFlags().StringVar(&o.Address, AddressFlagName, o.Address, addressFlagHelp)
}
