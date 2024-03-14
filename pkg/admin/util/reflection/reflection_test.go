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

package reflection

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	_ "dubbo.apache.org/dubbo-go/v3/imports"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/server"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	grpcreflection "google.golang.org/grpc/reflection"

	"github.com/apache/dubbo-kubernetes/pkg/admin/util/reflection/testdata/api"
	greetgrpc "github.com/apache/dubbo-kubernetes/pkg/admin/util/reflection/testdata/proto/grpc_gen"
	"github.com/apache/dubbo-kubernetes/pkg/admin/util/reflection/testdata/proto/triple_gen/greettriple"
)

const (
	grpcPort   = 50086
	triplePort = 50087
	host       = "127.0.0.1"
)

var (
	tripleServer *server.Server
	grpcServer   *grpc.Server
	grpcRef      RPCReflection
	tripleRef    RPCReflection
)

func TestMain(m *testing.M) {
	initGRpcServer()
	initTripleServer()
	initGrpcReflection()
	initTripleReflection()
	os.Exit(m.Run())
}

func initGRpcServer() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		panic(err)
	}

	grpcServer = grpc.NewServer()
	greetgrpc.RegisterGreetServiceServer(grpcServer, &api.GreetGRPCServer{})
	grpcreflection.Register(grpcServer)
	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			panic(err)
		}
	}()
}

func initTripleServer() {
	if tripleServer != nil {
		return
	}

	var err error
	tripleServer, err = server.NewServer(
		server.WithServerProtocol(
			protocol.WithTriple(),
			protocol.WithPort(triplePort),
		),
	)
	if err != nil {
		panic(err)
	}

	if err = greettriple.RegisterGreetServiceHandler(tripleServer, &api.GreetTripleServer{}); err != nil {
		panic(err)
	}

	go func() {
		if err = tripleServer.Serve(); err != nil {
			panic(err)
		}
	}()
}

func initGrpcReflection() {
	target := fmt.Sprintf("%v:%v", host, grpcPort)
	grpcRef = NewRPCReflection(target)

	maxTry := 3
	var err error
	for i := 0; i < maxTry; i++ {
		err = grpcRef.Dail(context.Background())
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
	}
	if err != nil {
		panic(err)
	}
}

func initTripleReflection() {
	target := fmt.Sprintf("%v:%v", host, triplePort)
	tripleRef = NewRPCReflection(target)

	maxTry := 3
	var err error
	for i := 0; i < maxTry; i++ {
		err = tripleRef.Dail(context.Background())
		if err == nil {
			break
		}
		time.Sleep(time.Second * 5)
	}
	if err != nil {
		panic(err)
	}
}

func TestListServices(t *testing.T) {
	lists := []struct {
		name     string
		ref      RPCReflection
		services []string
	}{
		{
			name:     "grpc",
			ref:      grpcRef,
			services: []string{"greet.GreetService", "grpc.reflection.v1.ServerReflection", "grpc.reflection.v1alpha.ServerReflection"},
		},
		{
			name:     "triple",
			ref:      tripleRef,
			services: []string{"greet.GreetService", "grpc.health.v1.Health", "grpc.reflection.v1alpha.ServerReflection"},
		},
	}

	for _, list := range lists {
		t.Run(list.name, func(t *testing.T) {
			services, err := list.ref.ListServices()
			if err != nil {
				t.Error(err)
				return
			}

			t.Logf("name:%#v services:%#v", list.name, services)
			assert.Equal(t, services, list.services)
		})
	}
}

func TestListMethods(t *testing.T) {
	lists := []struct {
		name    string
		ref     RPCReflection
		service string
		methods []string
	}{
		{
			name:    "grpc",
			ref:     grpcRef,
			service: "greet.GreetService",
			methods: []string{"greet.GreetService.Greet", "greet.GreetService.GreetClientStream", "greet.GreetService.GreetServerStream", "greet.GreetService.GreetStream"},
		},
		{
			name:    "triple",
			ref:     tripleRef,
			service: "greet.GreetService",
			methods: []string{"greet.GreetService.Greet", "greet.GreetService.GreetClientStream", "greet.GreetService.GreetServerStream", "greet.GreetService.GreetStream"},
		},
	}

	for _, list := range lists {
		t.Run(list.name, func(t *testing.T) {
			methods, err := list.ref.ListMethods(list.service)
			if err != nil {
				t.Error(err)
				return
			}

			t.Logf("name:%#v service:%#v methods:%#v", list.name, list.service, methods)
			assert.Equal(t, methods, list.methods)
		})
	}
}

func TestMethodDescribeString(t *testing.T) {
	lists := []struct {
		name    string
		ref     RPCReflection
		service string
		methods []string
	}{
		{
			name:    "grpc",
			ref:     grpcRef,
			service: "greet.GreetService",
			methods: []string{"greet.GreetService.Greet", "greet.GreetService.GreetClientStream", "greet.GreetService.GreetServerStream", "greet.GreetService.GreetStream"},
		},
		{
			name:    "triple",
			ref:     tripleRef,
			service: "greet.GreetService",
			methods: []string{"greet.GreetService.Greet", "greet.GreetService.GreetClientStream", "greet.GreetService.GreetServerStream", "greet.GreetService.GreetStream"},
		},
	}
	var results []string

	for _, list := range lists {
		t.Run(list.name, func(t *testing.T) {
			var result string
			for _, method := range list.methods {
				txt, err := list.ref.DescribeString(method)
				if err != nil {
					t.Errorf("error in DescribeString for %v", method)
					return
				}
				t.Logf("name:%#v method:%#v txt:\n%#v", list.name, method, txt)
				result = result + txt
			}

			if len(results) > 0 {
				assert.Equal(t, results[0], result)
			} else {
				results = append(results, result)
			}
		})
	}
}

func TestInputAndOutputType(t *testing.T) {
	lists := []struct {
		name    string
		ref     RPCReflection
		service string
		methods []string
	}{
		{
			name:    "grpc",
			ref:     grpcRef,
			service: "greet.GreetService",
			methods: []string{"greet.GreetService.Greet", "greet.GreetService.GreetClientStream", "greet.GreetService.GreetServerStream", "greet.GreetService.GreetStream"},
		},
		{
			name:    "triple",
			ref:     tripleRef,
			service: "greet.GreetService",
			methods: []string{"greet.GreetService.Greet", "greet.GreetService.GreetClientStream", "greet.GreetService.GreetServerStream", "greet.GreetService.GreetStream"},
		},
	}
	var results []string

	for _, list := range lists {
		t.Run(list.name, func(t *testing.T) {
			var result string
			for _, method := range list.methods {

				inputType, outputType, err := list.ref.InputAndOutputType(method)
				if err != nil {
					t.Errorf("error in InputAndOutputType for %v", method)
					return
				}
				t.Logf("name:%#v method:%#v inputType:%#v outputType:%#v", list.name, method, inputType, outputType)

				result = inputType + outputType
			}

			if len(results) > 0 {
				assert.Equal(t, results[0], result)
			} else {
				results = append(results, result)
			}
		})
	}
}

func TestMessageDescribeString(t *testing.T) {
	lists := []struct {
		name    string
		ref     RPCReflection
		service string
		methods []string
	}{
		{
			name:    "grpc",
			ref:     grpcRef,
			service: "greet.GreetService",
			methods: []string{"greet.GreetService.Greet", "greet.GreetService.GreetClientStream", "greet.GreetService.GreetServerStream", "greet.GreetService.GreetStream"},
		},
		{
			name:    "triple",
			ref:     tripleRef,
			service: "greet.GreetService",
			methods: []string{"greet.GreetService.Greet", "greet.GreetService.GreetClientStream", "greet.GreetService.GreetServerStream", "greet.GreetService.GreetStream"},
		},
	}
	var results []string

	for _, list := range lists {
		t.Run(list.name, func(t *testing.T) {
			var result string

			for _, method := range list.methods {

				inputType, outputType, err := list.ref.InputAndOutputType(method)
				if err != nil {
					t.Errorf("error int DescribeString for %v", method)
					return
				}

				txt, err := list.ref.DescribeString(inputType)
				if err != nil {
					t.Errorf("error in DescribeString for %v", inputType)
					return
				}
				t.Logf("name:%#v inputType:%#v txt:\n%#v", list.name, inputType, txt)
				result = result + txt

				txt, err = list.ref.DescribeString(outputType)
				if err != nil {
					t.Errorf("error in DescribeString for %v", outputType)
					return
				}
				t.Logf("name:%#v outputType:%#v txt:\n%#v", list.name, outputType, txt)

				result = result + txt
			}

			if len(results) > 0 {
				assert.Equal(t, results[0], result)
			} else {
				results = append(results, result)
			}
		})
	}
}

func TestTemplateString(t *testing.T) {
	lists := []struct {
		name    string
		ref     RPCReflection
		service string
		methods []string
	}{
		{
			name:    "grpc",
			ref:     grpcRef,
			service: "greet.GreetService",
			methods: []string{"greet.GreetService.Greet", "greet.GreetService.GreetClientStream", "greet.GreetService.GreetServerStream", "greet.GreetService.GreetStream"},
		},
		{
			name:    "triple",
			ref:     tripleRef,
			service: "greet.GreetService",
			methods: []string{"greet.GreetService.Greet", "greet.GreetService.GreetClientStream", "greet.GreetService.GreetServerStream", "greet.GreetService.GreetStream"},
		},
	}
	var results []string

	for _, list := range lists {
		t.Run(list.name, func(t *testing.T) {
			var result string

			for _, method := range list.methods {

				inputType, outputType, err := list.ref.InputAndOutputType(method)
				if err != nil {
					t.Errorf("error in InputAndOutputType for %v", method)
					return
				}

				txt, err := list.ref.TemplateString(inputType)
				if err != nil {
					t.Errorf("error in TemplateString for %v", inputType)
					return
				}
				t.Logf("name:%#v inputType:%#v txt:\n%#v", list.name, inputType, txt)
				result = result + txt

				txt, err = list.ref.TemplateString(outputType)
				if err != nil {
					t.Errorf("error in TemplateString for %v", outputType)
					return
				}
				t.Logf("name:%#v outputType:%#v txt:\n%#v", list.name, outputType, txt)
				result = result + txt
			}

			if len(results) > 0 {
				assert.Equal(t, results[0], result)
			} else {
				results = append(results, result)
			}
		})
	}
}

func TestInvokeUnary(t *testing.T) {
	lists := []struct {
		name    string
		ref     RPCReflection
		service string
		method  string
		input   string
	}{
		{
			name:    "grpc",
			ref:     grpcRef,
			service: "greet.GreetService",
			method:  "greet.GreetService.Greet",
			input:   "{\n  \"name\": \"dubbo-kubernetes\"\n}",
		},
		{
			name:    "triple",
			ref:     tripleRef,
			service: "greet.GreetService",
			method:  "greet.GreetService.Greet",
			input:   "{\n  \"name\": \"dubbo-kubernetes\"\n}",
		},
	}
	var results []string

	for _, list := range lists {
		t.Run(list.name, func(t *testing.T) {
			resp, err := list.ref.Invoke(context.Background(), list.method, list.input)
			if err != nil {
				t.Errorf("error int Invoke for %v, err=%v", list.method, err)
				return
			}

			t.Logf("name:%#v method:%#v invoke success, response:\n%#v", list.name, list.method, resp)

			if len(results) > 0 {
				assert.Equal(t, results[0], resp)
			} else {
				results = append(results, resp)
			}
		})
	}
}

func TestInvokeServerStream(t *testing.T) {
	lists := []struct {
		name    string
		ref     RPCReflection
		service string
		method  string
		input   string
	}{
		{
			name:    "grpc",
			ref:     grpcRef,
			service: "greet.GreetService",
			method:  "greet.GreetService.GreetServerStream",
			input:   "{\n  \"name\": \"dubbo-kubernetes\"\n}",
		},
		{
			name:    "triple",
			ref:     tripleRef,
			service: "greet.GreetService",
			method:  "greet.GreetService.GreetServerStream",
			input:   "{\n  \"name\": \"dubbo-kubernetes\"\n}",
		},
	}
	var results []string

	for _, list := range lists {
		t.Run(list.name, func(t *testing.T) {
			resp, err := list.ref.Invoke(context.Background(), list.method, list.input)
			if err != nil {
				t.Errorf("error int Invoke for %v, err=%v", list.method, err)
				return
			}

			t.Logf("name:%#v method:%#v invoke success, response:\n%#v", list.name, list.method, resp)

			if len(results) > 0 {
				assert.Equal(t, results[0], resp)
			} else {
				results = append(results, resp)
			}
		})
	}
}

func TestInvokeClientStream(t *testing.T) {
	lists := []struct {
		name    string
		ref     RPCReflection
		service string
		method  string
		input   string
	}{
		{
			name:    "grpc",
			ref:     grpcRef,
			service: "greet.GreetService",
			method:  "greet.GreetService.GreetClientStream",
			input:   "{\n  \"name\": \"dubbo-kubernetes-1\"\n}" + "{\n  \"name\": \"dubbo-kubernetes-2\"\n}",
		},
		{
			name:    "triple",
			ref:     tripleRef,
			service: "greet.GreetService",
			method:  "greet.GreetService.GreetClientStream",
			input:   "{\n  \"name\": \"dubbo-kubernetes-1\"\n}" + "{\n  \"name\": \"dubbo-kubernetes-2\"\n}",
		},
	}
	var results []string

	for _, list := range lists {
		t.Run(list.name, func(t *testing.T) {
			resp, err := list.ref.Invoke(context.Background(), list.method, list.input)
			if err != nil {
				t.Errorf("error int Invoke for %v, err=%v", list.method, err)
				return
			}

			t.Logf("name:%#v method:%#v invoke success, response:\n%#v", list.name, list.method, resp)

			if len(results) > 0 {
				assert.Equal(t, results[0], resp)
			} else {
				results = append(results, resp)
			}
		})
	}
}

func TestInvokeBiStream(t *testing.T) {
	lists := []struct {
		name    string
		ref     RPCReflection
		service string
		method  string
		input   string
	}{
		{
			name:    "grpc",
			ref:     grpcRef,
			service: "greet.GreetService",
			method:  "greet.GreetService.GreetStream",
			input:   "{\n  \"name\": \"dubbo-kubernetes-1\"\n}" + "{\n  \"name\": \"dubbo-kubernetes-2\"\n}",
		},
		{
			name:    "triple",
			ref:     tripleRef,
			service: "greet.GreetService",
			method:  "greet.GreetService.GreetStream",
			input:   "{\n  \"name\": \"dubbo-kubernetes-1\"\n}" + "{\n  \"name\": \"dubbo-kubernetes-2\"\n}",
		},
	}
	var results []string

	for _, list := range lists {
		t.Run(list.name, func(t *testing.T) {
			resp, err := list.ref.Invoke(context.Background(), list.method, list.input)
			if err != nil {
				t.Errorf("error int Invoke for %v, err=%v", list.method, err)
				return
			}

			t.Logf("name:%#v method:%#v invoke success, response:\n%#v", list.name, list.method, resp)

			if len(results) > 0 {
				assert.Equal(t, results[0], resp)
			} else {
				results = append(results, resp)
			}
		})
	}
}
